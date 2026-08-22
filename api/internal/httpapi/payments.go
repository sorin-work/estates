package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type paymentStore interface {
	ListPayments(ctx context.Context, ownerID string, f store.PaymentFilter) ([]store.Payment, int, error)
	GetPayment(ctx context.Context, ownerID, id string) (store.Payment, error)
	CreatePayment(ctx context.Context, ownerID, leaseID string, in store.PaymentInput) (string, error)
	DeletePayment(ctx context.Context, ownerID, id string) error
	LedgerEntries(ctx context.Context, ownerID, leaseID string) (map[string]*store.LedgerEntries, error)
}

func (s *Server) paymentRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listPayments))
	r.Get("/export.csv", s.handle(s.exportPayments))
	r.Get("/{id}/receipt", s.handle(s.paymentReceipt))
	r.Delete("/{id}", s.handle(s.deletePayment))
}

type paymentList struct {
	Payments   []store.Payment `json:"payments"`
	Total      int             `json:"total"`
	TotalCents int64           `json:"page_total_cents"`
}

func paymentFilter(r *http.Request) (store.PaymentFilter, error) {
	var f store.PaymentFilter
	var err error
	if f.LeaseID, err = queryID(r, "lease_id"); err != nil {
		return f, err
	}
	if f.PropertyID, err = queryID(r, "property_id"); err != nil {
		return f, err
	}
	if f.From, err = queryDate(r, "from"); err != nil {
		return f, err
	}
	if f.To, err = queryDate(r, "to"); err != nil {
		return f, err
	}
	if f.Limit, err = queryInt(r, "limit", 50); err != nil {
		return f, err
	}
	if f.Offset, err = queryInt(r, "offset", 0); err != nil {
		return f, err
	}
	return f, nil
}

func (s *Server) listPayments(w http.ResponseWriter, r *http.Request) error {
	f, err := paymentFilter(r)
	if err != nil {
		return err
	}
	payments, total, err := s.store.ListPayments(r.Context(), userID(r), f)
	if err != nil {
		return err
	}
	resp := paymentList{Payments: payments, Total: total}
	for _, p := range payments {
		resp.TotalCents += p.AmountCents
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

func (s *Server) createPayment(w http.ResponseWriter, r *http.Request) error {
	leaseID, err := pathID(r, "lease")
	if err != nil {
		return err
	}
	var in store.PaymentInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	trim(&in.Method, &in.Reference, &in.Notes)
	if in.Method == "" {
		in.Method = "bank_transfer"
	}

	var v validator
	v.check(in.AmountCents > 0, "amount_cents", "must be more than zero")
	v.check(!in.PaidOn.IsZero(), "paid_on", "is required")
	v.check(!in.PaidOn.After(s.today()), "paid_on", "cannot be in the future")
	v.check(domain.OneOf(in.Method, domain.PaymentMethods), "method", "is not a known payment method")
	v.maxLen(in.Reference, "reference", 100)
	v.maxLen(in.Notes, "notes", 1000)
	if err := v.err(); err != nil {
		return err
	}

	id, err := s.store.CreatePayment(r.Context(), userID(r), leaseID, in)
	if err != nil {
		return notFoundAs(err, "lease")
	}
	p, err := s.store.GetPayment(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, p)
	return nil
}

func (s *Server) deletePayment(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "payment")
	if err != nil {
		return err
	}
	if err := s.store.DeletePayment(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "payment")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// exportPayments sends every payment matching the filter as CSV, for
// bookkeeping and tax returns.
func (s *Server) exportPayments(w http.ResponseWriter, r *http.Request) error {
	f, err := paymentFilter(r)
	if err != nil {
		return err
	}
	f.Limit, f.Offset = store.ExportAll, 0
	payments, _, err := s.store.ListPayments(r.Context(), userID(r), f)
	if err != nil {
		return err
	}
	rows := make([][]string, len(payments))
	for i, p := range payments {
		rows[i] = []string{p.PaidOn.String(), p.TenantName, p.PropertyName, p.UnitLabel,
			csvMoney(p.AmountCents), p.Method, p.Reference, p.Notes}
	}
	return writeCSV(w, "payments.csv",
		[]string{"Date", "Tenant", "Property", "Unit", "Amount", "Method", "Reference", "Notes"}, rows)
}
