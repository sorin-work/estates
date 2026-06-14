package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type vendorStore interface {
	ListVendors(ctx context.Context, ownerID, trade string) ([]store.Vendor, error)
	GetVendor(ctx context.Context, ownerID, id string) (store.Vendor, error)
	CreateVendor(ctx context.Context, ownerID string, in store.VendorInput) (string, error)
	UpdateVendor(ctx context.Context, ownerID, id string, in store.VendorInput) error
	DeleteVendor(ctx context.Context, ownerID, id string) error
}

func (s *Server) vendorRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listVendors))
	r.Post("/", s.handle(s.createVendor))
	r.Get("/{id}", s.handle(s.getVendor))
	r.Put("/{id}", s.handle(s.updateVendor))
	r.Delete("/{id}", s.handle(s.deleteVendor))
}

func (s *Server) listVendors(w http.ResponseWriter, r *http.Request) error {
	trade := r.URL.Query().Get("trade")
	if trade != "" && !domain.OneOf(trade, domain.VendorTrades) {
		return errBadRequest("unknown trade " + trade)
	}
	vendors, err := s.store.ListVendors(r.Context(), userID(r), trade)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, vendors)
	return nil
}

func (s *Server) getVendor(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "vendor")
	if err != nil {
		return err
	}
	v, err := s.store.GetVendor(r.Context(), userID(r), id)
	if err != nil {
		return notFoundAs(err, "vendor")
	}
	writeJSON(w, http.StatusOK, v)
	return nil
}

func decodeVendor(w http.ResponseWriter, r *http.Request) (store.VendorInput, error) {
	var in store.VendorInput
	if err := decode(w, r, &in); err != nil {
		return in, err
	}
	trim(&in.Name, &in.Trade, &in.ContactName, &in.Phone, &in.Email, &in.Notes)
	in.Email = strings.ToLower(in.Email)
	if in.Trade == "" {
		in.Trade = "other"
	}

	var v validator
	v.required(in.Name, "name")
	v.maxLen(in.Name, "name", 120)
	v.check(domain.OneOf(in.Trade, domain.VendorTrades), "trade", "is not a known trade")
	v.maxLen(in.ContactName, "contact_name", 120)
	v.maxLen(in.Phone, "phone", 40)
	v.email(in.Email, "email")
	v.maxLen(in.Notes, "notes", 5000)
	return in, v.err()
}

func (s *Server) createVendor(w http.ResponseWriter, r *http.Request) error {
	in, err := decodeVendor(w, r)
	if err != nil {
		return err
	}
	id, err := s.store.CreateVendor(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	v, err := s.store.GetVendor(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, v)
	return nil
}

func (s *Server) updateVendor(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "vendor")
	if err != nil {
		return err
	}
	in, err := decodeVendor(w, r)
	if err != nil {
		return err
	}
	if err := s.store.UpdateVendor(r.Context(), userID(r), id, in); err != nil {
		return notFoundAs(err, "vendor")
	}
	v, err := s.store.GetVendor(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, v)
	return nil
}

func (s *Server) deleteVendor(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "vendor")
	if err != nil {
		return err
	}
	if err := s.store.DeleteVendor(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "vendor")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
