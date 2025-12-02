package httpapi

import (
	"context"
	"math"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type unitStore interface {
	ListUnits(ctx context.Context, ownerID, propertyID string, asOf domain.Date) ([]store.Unit, error)
	GetUnit(ctx context.Context, ownerID, id string, asOf domain.Date) (store.Unit, error)
	CreateUnit(ctx context.Context, ownerID, propertyID string, in store.UnitInput) (string, error)
	UpdateUnit(ctx context.Context, ownerID, id string, in store.UnitInput) error
	DeleteUnit(ctx context.Context, ownerID, id string) error
}

func (s *Server) unitRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listUnits))
	r.Get("/{id}", s.handle(s.getUnit))
	r.Put("/{id}", s.handle(s.updateUnit))
	r.Delete("/{id}", s.handle(s.deleteUnit))
}

func (s *Server) listUnits(w http.ResponseWriter, r *http.Request) error {
	propertyID, err := queryID(r, "property_id")
	if err != nil {
		return err
	}
	units, err := s.store.ListUnits(r.Context(), userID(r), propertyID, s.today())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, units)
	return nil
}

func (s *Server) listPropertyUnits(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "property")
	if err != nil {
		return err
	}
	if _, err := s.store.GetProperty(r.Context(), userID(r), id, s.today()); err != nil {
		return notFoundAs(err, "property")
	}
	units, err := s.store.ListUnits(r.Context(), userID(r), id, s.today())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, units)
	return nil
}

func (s *Server) getUnit(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "unit")
	if err != nil {
		return err
	}
	u, err := s.store.GetUnit(r.Context(), userID(r), id, s.today())
	if err != nil {
		return notFoundAs(err, "unit")
	}
	writeJSON(w, http.StatusOK, u)
	return nil
}

func decodeUnit(w http.ResponseWriter, r *http.Request) (store.UnitInput, error) {
	var in store.UnitInput
	if err := decode(w, r, &in); err != nil {
		return in, err
	}
	trim(&in.Label)

	var v validator
	v.required(in.Label, "label")
	v.maxLen(in.Label, "label", 40)
	v.check(in.Bedrooms >= 0 && in.Bedrooms <= 50, "bedrooms", "must be between 0 and 50")
	v.check(in.Bathrooms >= 0 && in.Bathrooms <= 50, "bathrooms", "must be between 0 and 50")
	v.check(math.Mod(in.Bathrooms*2, 1) == 0, "bathrooms", "must be a whole or half number")
	v.check(in.SquareFeet == nil || *in.SquareFeet > 0, "square_feet", "must be positive")
	v.check(in.MarketRentCents >= 0, "market_rent_cents", "cannot be negative")
	return in, v.err()
}

func (s *Server) createUnit(w http.ResponseWriter, r *http.Request) error {
	propertyID, err := pathID(r, "property")
	if err != nil {
		return err
	}
	in, err := decodeUnit(w, r)
	if err != nil {
		return err
	}
	id, err := s.store.CreateUnit(r.Context(), userID(r), propertyID, in)
	if err != nil {
		return notFoundAs(err, "property")
	}
	u, err := s.store.GetUnit(r.Context(), userID(r), id, s.today())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, u)
	return nil
}

func (s *Server) updateUnit(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "unit")
	if err != nil {
		return err
	}
	in, err := decodeUnit(w, r)
	if err != nil {
		return err
	}
	if err := s.store.UpdateUnit(r.Context(), userID(r), id, in); err != nil {
		return notFoundAs(err, "unit")
	}
	u, err := s.store.GetUnit(r.Context(), userID(r), id, s.today())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, u)
	return nil
}

func (s *Server) deleteUnit(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "unit")
	if err != nil {
		return err
	}
	if err := s.store.DeleteUnit(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "unit")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
