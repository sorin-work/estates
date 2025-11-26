package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type propertyStore interface {
	ListProperties(ctx context.Context, ownerID string) ([]store.Property, error)
	GetProperty(ctx context.Context, ownerID, id string) (store.Property, error)
	CreateProperty(ctx context.Context, ownerID string, in store.PropertyInput) (string, error)
	UpdateProperty(ctx context.Context, ownerID, id string, in store.PropertyInput) error
	DeleteProperty(ctx context.Context, ownerID, id string) error
}

func (s *Server) propertyRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listProperties))
	r.Post("/", s.handle(s.createProperty))
	r.Get("/{id}", s.handle(s.getProperty))
	r.Put("/{id}", s.handle(s.updateProperty))
	r.Delete("/{id}", s.handle(s.deleteProperty))
	r.Get("/{id}/units", s.handle(s.listPropertyUnits))
	r.Post("/{id}/units", s.handle(s.createUnit))
}

func (s *Server) listProperties(w http.ResponseWriter, r *http.Request) error {
	props, err := s.store.ListProperties(r.Context(), userID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, props)
	return nil
}

func (s *Server) getProperty(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "property")
	if err != nil {
		return err
	}
	p, err := s.store.GetProperty(r.Context(), userID(r), id)
	if err != nil {
		return notFoundAs(err, "property")
	}
	writeJSON(w, http.StatusOK, p)
	return nil
}

func decodeProperty(w http.ResponseWriter, r *http.Request) (store.PropertyInput, error) {
	var in store.PropertyInput
	if err := decode(w, r, &in); err != nil {
		return in, err
	}
	trim(&in.Name, &in.Kind, &in.AddressLine1, &in.AddressLine2, &in.City, &in.Region, &in.PostalCode)
	in.Notes = strings.TrimSpace(in.Notes)
	if in.Kind == "" {
		in.Kind = "single_family"
	}

	var v validator
	v.required(in.Name, "name")
	v.maxLen(in.Name, "name", 120)
	v.check(domain.OneOf(in.Kind, domain.PropertyKinds), "kind", "is not a known property type")
	v.required(in.AddressLine1, "address_line1")
	v.required(in.City, "city")
	v.maxLen(in.Notes, "notes", 5000)
	return in, v.err()
}

func (s *Server) createProperty(w http.ResponseWriter, r *http.Request) error {
	in, err := decodeProperty(w, r)
	if err != nil {
		return err
	}
	id, err := s.store.CreateProperty(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	p, err := s.store.GetProperty(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, p)
	return nil
}

func (s *Server) updateProperty(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "property")
	if err != nil {
		return err
	}
	in, err := decodeProperty(w, r)
	if err != nil {
		return err
	}
	if err := s.store.UpdateProperty(r.Context(), userID(r), id, in); err != nil {
		return notFoundAs(err, "property")
	}
	p, err := s.store.GetProperty(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, p)
	return nil
}

func (s *Server) deleteProperty(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "property")
	if err != nil {
		return err
	}
	if err := s.store.DeleteProperty(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "property")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
