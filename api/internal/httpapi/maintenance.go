package httpapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type maintenanceStore interface {
	ListMaintenance(ctx context.Context, ownerID string, f store.MaintenanceFilter) ([]store.MaintenanceRequest, error)
	GetMaintenance(ctx context.Context, ownerID, id string) (store.MaintenanceRequest, error)
	CreateMaintenance(ctx context.Context, ownerID string, in store.MaintenanceInput) (string, error)
	UpdateMaintenance(ctx context.Context, ownerID, id string, in store.MaintenanceInput) error
	SetMaintenanceStatus(ctx context.Context, ownerID, id, status string, completedOn *domain.Date) error
	DeleteMaintenance(ctx context.Context, ownerID, id string) error
}

func (s *Server) maintenanceRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listMaintenance))
	r.Post("/", s.handle(s.createMaintenance))
	r.Get("/{id}", s.handle(s.getMaintenance))
	r.Put("/{id}", s.handle(s.updateMaintenance))
	r.Post("/{id}/status", s.handle(s.setMaintenanceStatus))
	r.Delete("/{id}", s.handle(s.deleteMaintenance))
}

// maintenanceView adds the statuses the request can move to, so the web app
// does not have to duplicate the workflow rules.
type maintenanceView struct {
	store.MaintenanceRequest
	NextStatuses []string `json:"next_statuses"`
}

func viewMaintenance(m store.MaintenanceRequest) maintenanceView {
	next := domain.NextMaintenanceStatuses(m.Status)
	if next == nil {
		next = []string{}
	}
	return maintenanceView{MaintenanceRequest: m, NextStatuses: next}
}

func (s *Server) listMaintenance(w http.ResponseWriter, r *http.Request) error {
	var f store.MaintenanceFilter
	var err error
	if f.PropertyID, err = queryID(r, "property_id"); err != nil {
		return err
	}
	f.Status = r.URL.Query().Get("status")
	if f.Status != "" && f.Status != "active" && !domain.OneOf(f.Status, domain.MaintenanceStatuses) {
		return errBadRequest("unknown status " + f.Status)
	}
	f.Priority = r.URL.Query().Get("priority")
	if f.Priority != "" && !domain.OneOf(f.Priority, domain.MaintenancePriorities) {
		return errBadRequest("unknown priority " + f.Priority)
	}

	list, err := s.store.ListMaintenance(r.Context(), userID(r), f)
	if err != nil {
		return err
	}
	out := make([]maintenanceView, len(list))
	for i, m := range list {
		out[i] = viewMaintenance(m)
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) getMaintenance(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "maintenance request")
	if err != nil {
		return err
	}
	m, err := s.store.GetMaintenance(r.Context(), userID(r), id)
	if err != nil {
		return notFoundAs(err, "maintenance request")
	}
	writeJSON(w, http.StatusOK, viewMaintenance(m))
	return nil
}

func (s *Server) decodeMaintenance(w http.ResponseWriter, r *http.Request) (store.MaintenanceInput, error) {
	var in store.MaintenanceInput
	if err := decode(w, r, &in); err != nil {
		return in, err
	}
	trim(&in.PropertyID, &in.Title, &in.Description, &in.Priority, &in.Vendor)
	if in.Priority == "" {
		in.Priority = "normal"
	}
	if in.ReportedOn.IsZero() {
		in.ReportedOn = s.today()
	}

	var v validator
	v.check(isUUID(in.PropertyID), "property_id", "is required")
	v.check(in.UnitID == nil || *in.UnitID == "" || isUUID(*in.UnitID), "unit_id", "is not a valid id")
	v.required(in.Title, "title")
	v.maxLen(in.Title, "title", 200)
	v.maxLen(in.Description, "description", 5000)
	v.check(domain.OneOf(in.Priority, domain.MaintenancePriorities), "priority", "must be low, normal, high or urgent")
	v.maxLen(in.Vendor, "vendor", 120)
	return in, v.err()
}

func (s *Server) createMaintenance(w http.ResponseWriter, r *http.Request) error {
	in, err := s.decodeMaintenance(w, r)
	if err != nil {
		return err
	}
	id, err := s.store.CreateMaintenance(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	m, err := s.store.GetMaintenance(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, viewMaintenance(m))
	return nil
}

func (s *Server) updateMaintenance(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "maintenance request")
	if err != nil {
		return err
	}
	in, err := s.decodeMaintenance(w, r)
	if err != nil {
		return err
	}
	if err := s.store.UpdateMaintenance(r.Context(), userID(r), id, in); err != nil {
		return notFoundAs(err, "maintenance request")
	}
	m, err := s.store.GetMaintenance(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, viewMaintenance(m))
	return nil
}

func (s *Server) setMaintenanceStatus(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "maintenance request")
	if err != nil {
		return err
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if !domain.OneOf(in.Status, domain.MaintenanceStatuses) {
		return &apiError{
			Status: http.StatusUnprocessableEntity, Code: "validation_failed", Message: "some fields are invalid",
			Fields: map[string]string{"status": "is not a known status"},
		}
	}

	m, err := s.store.GetMaintenance(r.Context(), userID(r), id)
	if err != nil {
		return notFoundAs(err, "maintenance request")
	}
	if m.Status == in.Status {
		writeJSON(w, http.StatusOK, viewMaintenance(m))
		return nil
	}
	if !domain.CanTransitionMaintenance(m.Status, in.Status) {
		return &apiError{
			Status:  http.StatusUnprocessableEntity,
			Code:    "invalid_transition",
			Message: fmt.Sprintf("a request that is %s cannot be moved to %s", label(m.Status), label(in.Status)),
		}
	}

	var completedOn *domain.Date
	if in.Status == domain.MaintenanceCompleted {
		today := s.today()
		completedOn = &today
	}
	if err := s.store.SetMaintenanceStatus(r.Context(), userID(r), id, in.Status, completedOn); err != nil {
		return notFoundAs(err, "maintenance request")
	}
	m, err = s.store.GetMaintenance(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, viewMaintenance(m))
	return nil
}

func (s *Server) deleteMaintenance(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "maintenance request")
	if err != nil {
		return err
	}
	if err := s.store.DeleteMaintenance(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "maintenance request")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// label turns a status like in_progress into "in progress" for messages.
func label(status string) string {
	out := []byte(status)
	for i, c := range out {
		if c == '_' {
			out[i] = ' '
		}
	}
	return string(out)
}
