package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/store"
)

type renewalStore interface {
	AddRentChange(ctx context.Context, ownerID, leaseID string, in store.RentChangeInput) (string, error)
	DeleteRentChange(ctx context.Context, ownerID, id string) error
	RenewLease(ctx context.Context, ownerID, leaseID string, in store.RenewalInput) (string, error)
}

func (s *Server) rentChangeRoutes(r chi.Router) {
	r.Delete("/{id}", s.handle(s.deleteRentChange))
}

// addRentChange records a rent increase. Rent already charged for periods
// from the new date is updated, and anything due is billed straight away.
func (s *Server) addRentChange(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "lease")
	if err != nil {
		return err
	}
	var in store.RentChangeInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	trim(&in.Notes)

	var v validator
	v.check(!in.EffectiveDate.IsZero(), "effective_date", "is required")
	v.check(in.RentCents > 0, "rent_cents", "must be more than zero")
	v.maxLen(in.Notes, "notes", 1000)
	if err := v.err(); err != nil {
		return err
	}
	if _, err := s.store.AddRentChange(r.Context(), userID(r), id, in); err != nil {
		return notFoundAs(err, "lease")
	}
	l, err := s.billNow(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	s.audit(r, "lease", l.ID, l.ID, "rent_changed", "Rent for "+l.TenantName+" set to "+money(in.RentCents)+" from "+in.EffectiveDate.String())
	writeJSON(w, http.StatusCreated, l)
	return nil
}

func (s *Server) deleteRentChange(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "rent change")
	if err != nil {
		return err
	}
	if err := s.store.DeleteRentChange(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "rent change")
	}
	s.audit(r, "rent_change", id, "", "deleted", "Removed a rent change")
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) renewLease(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "lease")
	if err != nil {
		return err
	}
	var in store.RenewalInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	trim(&in.Notes)

	var v validator
	v.check(!in.EndDate.IsZero(), "end_date", "is required")
	v.check(in.RentCents > 0, "rent_cents", "must be more than zero")
	v.maxLen(in.Notes, "notes", 5000)
	if err := v.err(); err != nil {
		return err
	}
	newID, err := s.store.RenewLease(r.Context(), userID(r), id, in)
	if err != nil {
		return notFoundAs(err, "lease")
	}
	l, err := s.billNow(r.Context(), userID(r), newID)
	if err != nil {
		return err
	}
	s.audit(r, "lease", l.ID, id, "renewed", "Renewed the lease of "+l.TenantName+" until "+l.EndDate.String()+" at "+money(l.RentCents))
	s.audit(r, "lease", l.ID, l.ID, "created", "Renewal of an earlier lease, from "+l.StartDate.String())
	writeJSON(w, http.StatusCreated, l)
	return nil
}
