package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type leaseStore interface {
	ListLeases(ctx context.Context, ownerID string, f store.LeaseFilter) ([]store.Lease, error)
	GetLease(ctx context.Context, ownerID, id string) (store.Lease, error)
	CreateLease(ctx context.Context, ownerID string, in store.LeaseInput) (string, error)
	UpdateLease(ctx context.Context, ownerID, id string, in store.LeaseInput) error
	TerminateLease(ctx context.Context, ownerID, id string, on domain.Date) error
	DeleteLease(ctx context.Context, ownerID, id string) error
}

var leaseStatuses = []string{domain.LeaseUpcoming, domain.LeaseActive, domain.LeaseEnded, domain.LeaseTerminated}

func (s *Server) leaseRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listLeases))
	r.Post("/", s.handle(s.createLease))
	r.Get("/{id}", s.handle(s.getLease))
	r.Put("/{id}", s.handle(s.updateLease))
	r.Delete("/{id}", s.handle(s.deleteLease))
	r.Post("/{id}/terminate", s.handle(s.terminateLease))
}

func (s *Server) withStatus(l store.Lease) store.Lease {
	l.Status = domain.LeaseStatus(l.StartDate, l.EndDate, l.TerminatedOn, s.today())
	return l
}

func (s *Server) listLeases(w http.ResponseWriter, r *http.Request) error {
	var f store.LeaseFilter
	var err error
	if f.PropertyID, err = queryID(r, "property_id"); err != nil {
		return err
	}
	if f.UnitID, err = queryID(r, "unit_id"); err != nil {
		return err
	}
	if f.TenantID, err = queryID(r, "tenant_id"); err != nil {
		return err
	}
	status := r.URL.Query().Get("status")
	if status != "" && !domain.OneOf(status, leaseStatuses) {
		return errBadRequest("status must be one of " + strings.Join(leaseStatuses, ", "))
	}

	leases, err := s.store.ListLeases(r.Context(), userID(r), f)
	if err != nil {
		return err
	}
	out := make([]store.Lease, 0, len(leases))
	for _, l := range leases {
		l = s.withStatus(l)
		if status == "" || l.Status == status {
			out = append(out, l)
		}
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) loadLease(r *http.Request) (store.Lease, error) {
	id, err := pathID(r, "lease")
	if err != nil {
		return store.Lease{}, err
	}
	l, err := s.store.GetLease(r.Context(), userID(r), id)
	if err != nil {
		return store.Lease{}, notFoundAs(err, "lease")
	}
	return s.withStatus(l), nil
}

func (s *Server) getLease(w http.ResponseWriter, r *http.Request) error {
	l, err := s.loadLease(r)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, l)
	return nil
}

func decodeLease(w http.ResponseWriter, r *http.Request) (store.LeaseInput, error) {
	var in store.LeaseInput
	if err := decode(w, r, &in); err != nil {
		return in, err
	}
	trim(&in.UnitID, &in.TenantID, &in.Notes)
	if in.DueDay == 0 {
		in.DueDay = 1
	}

	var v validator
	v.check(isUUID(in.UnitID), "unit_id", "is required")
	v.check(isUUID(in.TenantID), "tenant_id", "is required")
	v.check(!in.StartDate.IsZero(), "start_date", "is required")
	v.check(!in.EndDate.IsZero(), "end_date", "is required")
	if !in.StartDate.IsZero() && !in.EndDate.IsZero() {
		v.check(!in.EndDate.Before(in.StartDate), "end_date", "must be on or after the start date")
	}
	v.check(in.RentCents > 0, "rent_cents", "must be more than zero")
	v.check(in.DepositCents >= 0, "deposit_cents", "cannot be negative")
	v.check(in.DueDay >= 1 && in.DueDay <= 31, "due_day", "must be between 1 and 31")
	v.check(in.GraceDays >= 0 && in.GraceDays <= 31, "grace_days", "must be between 0 and 31")
	v.check(in.LateFeeCents >= 0, "late_fee_cents", "cannot be negative")
	v.maxLen(in.Notes, "notes", 5000)
	return in, v.err()
}

// reload fetches a lease after a change, with its status.
func (s *Server) reload(ctx context.Context, ownerID, leaseID string) (store.Lease, error) {
	l, err := s.store.GetLease(ctx, ownerID, leaseID)
	if err != nil {
		return store.Lease{}, err
	}
	return s.withStatus(l), nil
}

func (s *Server) createLease(w http.ResponseWriter, r *http.Request) error {
	in, err := decodeLease(w, r)
	if err != nil {
		return err
	}
	id, err := s.store.CreateLease(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	l, err := s.reload(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, l)
	return nil
}

func (s *Server) updateLease(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "lease")
	if err != nil {
		return err
	}
	in, err := decodeLease(w, r)
	if err != nil {
		return err
	}
	if err := s.store.UpdateLease(r.Context(), userID(r), id, in); err != nil {
		return notFoundAs(err, "lease")
	}
	l, err := s.reload(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, l)
	return nil
}

func (s *Server) terminateLease(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "lease")
	if err != nil {
		return err
	}
	var in struct {
		Date domain.Date `json:"date"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	var v validator
	v.check(!in.Date.IsZero(), "date", "is required")
	if err := v.err(); err != nil {
		return err
	}
	if err := s.store.TerminateLease(r.Context(), userID(r), id, in.Date); err != nil {
		return notFoundAs(err, "lease")
	}
	l, err := s.store.GetLease(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, s.withStatus(l))
	return nil
}

func (s *Server) deleteLease(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "lease")
	if err != nil {
		return err
	}
	if err := s.store.DeleteLease(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "lease")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
