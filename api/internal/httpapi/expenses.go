package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type expenseStore interface {
	ListExpenses(ctx context.Context, ownerID string, f store.ExpenseFilter) ([]store.Expense, error)
	ExpenseTotals(ctx context.Context, ownerID string, f store.ExpenseFilter) ([]store.CategoryTotal, error)
	GetExpense(ctx context.Context, ownerID, id string) (store.Expense, error)
	CreateExpense(ctx context.Context, ownerID string, in store.ExpenseInput) (string, error)
	UpdateExpense(ctx context.Context, ownerID, id string, in store.ExpenseInput) error
	DeleteExpense(ctx context.Context, ownerID, id string) error
}

func (s *Server) expenseRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listExpenses))
	r.Get("/export.csv", s.handle(s.exportExpenses))
	r.Post("/", s.handle(s.createExpense))
	r.Get("/{id}", s.handle(s.getExpense))
	r.Put("/{id}", s.handle(s.updateExpense))
	r.Delete("/{id}", s.handle(s.deleteExpense))
}

type expenseList struct {
	Expenses   []store.Expense       `json:"expenses"`
	TotalCents int64                 `json:"total_cents"`
	ByCategory []store.CategoryTotal `json:"by_category"`
}

func expenseFilter(r *http.Request) (store.ExpenseFilter, error) {
	var f store.ExpenseFilter
	var err error
	if f.PropertyID, err = queryID(r, "property_id"); err != nil {
		return f, err
	}
	if f.VendorID, err = queryID(r, "vendor_id"); err != nil {
		return f, err
	}
	f.Category = r.URL.Query().Get("category")
	if f.Category != "" && !domain.OneOf(f.Category, domain.ExpenseCategories) {
		return f, errBadRequest("unknown category " + f.Category)
	}
	if f.From, err = queryDate(r, "from"); err != nil {
		return f, err
	}
	if f.To, err = queryDate(r, "to"); err != nil {
		return f, err
	}
	return f, nil
}

func (s *Server) listExpenses(w http.ResponseWriter, r *http.Request) error {
	f, err := expenseFilter(r)
	if err != nil {
		return err
	}
	expenses, err := s.store.ListExpenses(r.Context(), userID(r), f)
	if err != nil {
		return err
	}
	totals, err := s.store.ExpenseTotals(r.Context(), userID(r), f)
	if err != nil {
		return err
	}
	resp := expenseList{Expenses: expenses, ByCategory: totals}
	for _, t := range totals {
		resp.TotalCents += t.AmountCents
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

func (s *Server) getExpense(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "expense")
	if err != nil {
		return err
	}
	e, err := s.store.GetExpense(r.Context(), userID(r), id)
	if err != nil {
		return notFoundAs(err, "expense")
	}
	writeJSON(w, http.StatusOK, e)
	return nil
}

func decodeExpense(w http.ResponseWriter, r *http.Request) (store.ExpenseInput, error) {
	var in store.ExpenseInput
	if err := decode(w, r, &in); err != nil {
		return in, err
	}
	trim(&in.PropertyID, &in.Category, &in.Vendor, &in.Description)

	var v validator
	v.check(isUUID(in.PropertyID), "property_id", "is required")
	v.check(in.MaintenanceRequestID == nil || *in.MaintenanceRequestID == "" || isUUID(*in.MaintenanceRequestID),
		"maintenance_request_id", "is not a valid id")
	v.check(domain.OneOf(in.Category, domain.ExpenseCategories), "category", "is not a known category")
	v.check(in.AmountCents > 0, "amount_cents", "must be more than zero")
	v.check(!in.SpentOn.IsZero(), "spent_on", "is required")
	v.maxLen(in.Vendor, "vendor", 120)
	v.check(in.VendorID == nil || *in.VendorID == "" || isUUID(*in.VendorID), "vendor_id", "is not a valid id")
	v.maxLen(in.Description, "description", 1000)
	return in, v.err()
}

func (s *Server) createExpense(w http.ResponseWriter, r *http.Request) error {
	in, err := decodeExpense(w, r)
	if err != nil {
		return err
	}
	id, err := s.store.CreateExpense(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	e, err := s.store.GetExpense(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	s.audit(r, "expense", e.ID, "", "created", "Expense of "+money(e.AmountCents)+" for "+e.PropertyName+" ("+e.Category+")")
	writeJSON(w, http.StatusCreated, e)
	return nil
}

func (s *Server) updateExpense(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "expense")
	if err != nil {
		return err
	}
	in, err := decodeExpense(w, r)
	if err != nil {
		return err
	}
	if err := s.store.UpdateExpense(r.Context(), userID(r), id, in); err != nil {
		return notFoundAs(err, "expense")
	}
	e, err := s.store.GetExpense(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, e)
	return nil
}

func (s *Server) deleteExpense(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "expense")
	if err != nil {
		return err
	}
	if err := s.store.DeleteExpense(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "expense")
	}
	s.audit(r, "expense", id, "", "deleted", "Deleted an expense")
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) exportExpenses(w http.ResponseWriter, r *http.Request) error {
	f, err := expenseFilter(r)
	if err != nil {
		return err
	}
	expenses, err := s.store.ListExpenses(r.Context(), userID(r), f)
	if err != nil {
		return err
	}
	rows := make([][]string, len(expenses))
	for i, e := range expenses {
		job := ""
		if e.MaintenanceTitle != nil {
			job = *e.MaintenanceTitle
		}
		rows[i] = []string{e.SpentOn.String(), e.PropertyName, e.Category, e.Vendor, e.Description, job, csvMoney(e.AmountCents)}
	}
	return writeCSV(w, "expenses.csv",
		[]string{"Date", "Property", "Category", "Vendor", "Description", "Maintenance request", "Amount"}, rows)
}
