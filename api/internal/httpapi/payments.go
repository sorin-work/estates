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
}

func (s *Server) paymentRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listPayments))
	r.Delete("/{id}", s.handle(s.deletePayment))
}

type paymentList struct {
	Payments   []store.Payment `json:"payments"`
	Total      int             `json:"total"`
	TotalCents int64           `json:"page_total_cents"`
}

func (s *Server) listPayments(w http.ResponseWriter, r *http.Request) error {
	var f store.PaymentFilter
	var err error
	if f.LeaseID, err = queryID(r, "lease_id"); err != nil {
		return err
	}
	if f.PropertyID, err = queryID(r, "property_id"); err != nil {
		return err
	}
	if f.From, err = queryDate(r, "from"); err != nil {
		return err
	}
	if f.To, err = queryDate(r, "to"); err != nil {
		return err
	}
	if f.Limit, err = queryInt(r, "limit", 50); err != nil {
		return err
	}
	if f.Offset, err = queryInt(r, "offset", 0); err != nil {
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
