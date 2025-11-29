package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/store"
)

type tenantStore interface {
	ListTenants(ctx context.Context, ownerID, search string) ([]store.Tenant, error)
	GetTenant(ctx context.Context, ownerID, id string) (store.Tenant, error)
	CreateTenant(ctx context.Context, ownerID string, in store.TenantInput) (string, error)
	UpdateTenant(ctx context.Context, ownerID, id string, in store.TenantInput) error
	DeleteTenant(ctx context.Context, ownerID, id string) error
}

func (s *Server) tenantRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listTenants))
	r.Post("/", s.handle(s.createTenant))
	r.Get("/{id}", s.handle(s.getTenant))
	r.Put("/{id}", s.handle(s.updateTenant))
	r.Delete("/{id}", s.handle(s.deleteTenant))
}

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	tenants, err := s.store.ListTenants(r.Context(), userID(r), q)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, tenants)
	return nil
}

func (s *Server) getTenant(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "tenant")
	if err != nil {
		return err
	}
	t, err := s.store.GetTenant(r.Context(), userID(r), id)
	if err != nil {
		return notFoundAs(err, "tenant")
	}
	writeJSON(w, http.StatusOK, t)
	return nil
}

func decodeTenant(w http.ResponseWriter, r *http.Request) (store.TenantInput, error) {
	var in store.TenantInput
	if err := decode(w, r, &in); err != nil {
		return in, err
	}
	trim(&in.FirstName, &in.LastName, &in.Email, &in.Phone, &in.Notes)
	in.Email = strings.ToLower(in.Email)

	var v validator
	v.required(in.FirstName, "first_name")
	v.maxLen(in.FirstName, "first_name", 80)
	v.required(in.LastName, "last_name")
	v.maxLen(in.LastName, "last_name", 80)
	v.email(in.Email, "email")
	v.maxLen(in.Phone, "phone", 40)
	v.maxLen(in.Notes, "notes", 5000)
	return in, v.err()
}

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) error {
	in, err := decodeTenant(w, r)
	if err != nil {
		return err
	}
	id, err := s.store.CreateTenant(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	t, err := s.store.GetTenant(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, t)
	return nil
}

func (s *Server) updateTenant(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "tenant")
	if err != nil {
		return err
	}
	in, err := decodeTenant(w, r)
	if err != nil {
		return err
	}
	if err := s.store.UpdateTenant(r.Context(), userID(r), id, in); err != nil {
		return notFoundAs(err, "tenant")
	}
	t, err := s.store.GetTenant(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, t)
	return nil
}

func (s *Server) deleteTenant(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "tenant")
	if err != nil {
		return err
	}
	if err := s.store.DeleteTenant(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "tenant")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
