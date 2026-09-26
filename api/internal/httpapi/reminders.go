package httpapi

import (
	"net/http"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

// reminders lists what the landlord should deal with soon.
func (s *Server) reminders(w http.ResponseWriter, r *http.Request) error {
	ctx, owner := r.Context(), userID(r)
	u, err := s.store.UserByID(ctx, owner)
	if err != nil {
		return err
	}
	leases, err := s.store.ListLeases(ctx, owner, store.LeaseFilter{})
	if err != nil {
		return err
	}
	held, err := s.store.DepositsHeld(ctx, owner)
	if err != nil {
		return err
	}
	requests, err := s.store.ListMaintenance(ctx, owner, store.MaintenanceFilter{Status: "active"})
	if err != nil {
		return err
	}

	facts := make([]domain.LeaseFacts, len(leases))
	for i, l := range leases {
		facts[i] = domain.LeaseFacts{
			ID: l.ID, TenantName: l.TenantName, Place: leasePlace(l),
			Start: l.StartDate, End: l.EndDate, TerminatedOn: l.TerminatedOn,
			Renewed: l.RenewedByID != nil, DepositCents: l.DepositCents, DepositHeld: held[l.ID],
			Terms: l.Terms(),
		}
	}
	jobs := make([]domain.MaintenanceFacts, len(requests))
	for i, m := range requests {
		jobs[i] = domain.MaintenanceFacts{ID: m.ID, Title: m.Title, Priority: m.Priority, Status: m.Status, ReportedOn: m.ReportedOn}
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, domain.Reminders(s.today(), facts, jobs, u.DepositReturnDays))
	return nil
}
