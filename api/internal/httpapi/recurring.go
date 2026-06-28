package httpapi

import (
	"context"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type recurringStore interface {
	ListRecurringExpenses(ctx context.Context, ownerID, propertyID string) ([]store.RecurringExpense, error)
	GetRecurringExpense(ctx context.Context, ownerID, id string) (store.RecurringExpense, error)
	CreateRecurringExpense(ctx context.Context, ownerID string, in store.RecurringExpenseInput) (string, error)
	UpdateRecurringExpense(ctx context.Context, ownerID, id string, in store.RecurringExpenseInput) error
	DeleteRecurringExpense(ctx context.Context, ownerID, id string) error
}

func (s *Server) recurringRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listRecurring))
	r.Post("/", s.handle(s.createRecurring))
	r.Get("/{id}", s.handle(s.getRecurring))
	r.Put("/{id}", s.handle(s.updateRecurring))
	r.Delete("/{id}", s.handle(s.deleteRecurring))
}

// recurringView adds the date the expense is next due.
type recurringView struct {
	store.RecurringExpense
	NextOn *domain.Date `json:"next_on"`
}

func (s *Server) viewRecurring(r store.RecurringExpense) recurringView {
	after := s.today()
	if r.GeneratedThrough != nil && r.GeneratedThrough.After(after) {
		after = *r.GeneratedThrough
	}
	v := recurringView{RecurringExpense: r}
	if next := r.Recurrence().Next(after); !next.IsZero() {
		v.NextOn = &next
	}
	return v
}

func (s *Server) listRecurring(w http.ResponseWriter, r *http.Request) error {
	propertyID, err := queryID(r, "property_id")
	if err != nil {
		return err
	}
	list, err := s.store.ListRecurringExpenses(r.Context(), userID(r), propertyID)
	if err != nil {
		return err
	}
	out := make([]recurringView, len(list))
	for i, re := range list {
		out[i] = s.viewRecurring(re)
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) getRecurring(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "recurring expense")
	if err != nil {
		return err
	}
	re, err := s.store.GetRecurringExpense(r.Context(), userID(r), id)
	if err != nil {
		return notFoundAs(err, "recurring expense")
	}
	writeJSON(w, http.StatusOK, s.viewRecurring(re))
	return nil
}

func decodeRecurring(w http.ResponseWriter, r *http.Request) (store.RecurringExpenseInput, error) {
	var in store.RecurringExpenseInput
	if err := decode(w, r, &in); err != nil {
		return in, err
	}
	trim(&in.PropertyID, &in.Vendor, &in.Category, &in.Description)
	if in.EveryMonths == 0 {
		in.EveryMonths = 1
	}
	if in.EndDate != nil && in.EndDate.IsZero() {
		in.EndDate = nil
	}

	var v validator
	v.check(isUUID(in.PropertyID), "property_id", "is required")
	v.check(in.VendorID == nil || *in.VendorID == "" || isUUID(*in.VendorID), "vendor_id", "is not a valid id")
	v.maxLen(in.Vendor, "vendor", 120)
	v.check(domain.OneOf(in.Category, domain.ExpenseCategories), "category", "is not a known category")
	v.check(in.AmountCents > 0, "amount_cents", "must be more than zero")
	v.maxLen(in.Description, "description", 1000)
	v.check(in.DayOfMonth >= 1 && in.DayOfMonth <= 31, "day_of_month", "must be between 1 and 31")
	v.check(slices.Contains(domain.RecurrenceMonths, in.EveryMonths), "every_months", "must be 1, 3, 6 or 12")
	v.check(!in.StartDate.IsZero(), "start_date", "is required")
	if in.EndDate != nil && !in.StartDate.IsZero() {
		v.check(!in.EndDate.Before(in.StartDate), "end_date", "must be on or after the start date")
	}
	return in, v.err()
}

// generateRecurring creates the expenses that are already due, so a new
// recurring expense shows up in the expense list straight away.
func (s *Server) generateRecurring(ctx context.Context, ownerID, id string) (recurringView, error) {
	re, err := s.store.GetRecurringExpense(ctx, ownerID, id)
	if err != nil {
		return recurringView{}, err
	}
	if _, err := s.billing.GenerateRecurring(ctx, re, s.today()); err != nil {
		return recurringView{}, err
	}
	re, err = s.store.GetRecurringExpense(ctx, ownerID, id)
	if err != nil {
		return recurringView{}, err
	}
	return s.viewRecurring(re), nil
}

func (s *Server) createRecurring(w http.ResponseWriter, r *http.Request) error {
	in, err := decodeRecurring(w, r)
	if err != nil {
		return err
	}
	id, err := s.store.CreateRecurringExpense(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	v, err := s.generateRecurring(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, v)
	return nil
}

func (s *Server) updateRecurring(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "recurring expense")
	if err != nil {
		return err
	}
	in, err := decodeRecurring(w, r)
	if err != nil {
		return err
	}
	if err := s.store.UpdateRecurringExpense(r.Context(), userID(r), id, in); err != nil {
		return notFoundAs(err, "recurring expense")
	}
	v, err := s.generateRecurring(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, v)
	return nil
}

func (s *Server) deleteRecurring(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "recurring expense")
	if err != nil {
		return err
	}
	if err := s.store.DeleteRecurringExpense(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "recurring expense")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
