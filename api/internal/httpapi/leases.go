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
	ListCharges(ctx context.Context, ownerID, leaseID string) ([]store.Charge, error)
	CreateCharge(ctx context.Context, ownerID, leaseID string, in store.ChargeInput) (string, error)
	DeleteCharge(ctx context.Context, ownerID, id string) error
}

var leaseStatuses = []string{domain.LeaseUpcoming, domain.LeaseActive, domain.LeaseEnded, domain.LeaseTerminated}

func (s *Server) leaseRoutes(r chi.Router) {
	r.Get("/", s.handle(s.listLeases))
	r.Post("/", s.handle(s.createLease))
	r.Get("/{id}", s.handle(s.getLease))
	r.Put("/{id}", s.handle(s.updateLease))
	r.Delete("/{id}", s.handle(s.deleteLease))
	r.Post("/{id}/terminate", s.handle(s.terminateLease))
	r.Get("/{id}/ledger", s.handle(s.leaseLedger))
	r.Post("/{id}/charges", s.handle(s.createCharge))
	r.Post("/{id}/payments", s.handle(s.createPayment))
	r.Post("/{id}/rent-changes", s.handle(s.addRentChange))
	r.Post("/{id}/renew", s.handle(s.renewLease))
	r.Get("/{id}/deposit", s.handle(s.leaseDeposit))
	r.Post("/{id}/deposit", s.handle(s.createDeposit))
}

func (s *Server) chargeRoutes(r chi.Router) {
	r.Delete("/{id}", s.handle(s.deleteCharge))
}

func (s *Server) withStatus(l store.Lease) store.Lease {
	l.Status = domain.LeaseStatus(l.StartDate, l.EndDate, l.TerminatedOn, s.today())
	l.CurrentRentCents = l.Terms().RentOn(s.today())
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

// billNow generates the rent a lease is due so far, so a new or edited lease
// shows the right balance without waiting for the background job.
func (s *Server) billNow(ctx context.Context, ownerID, leaseID string) (store.Lease, error) {
	l, err := s.store.GetLease(ctx, ownerID, leaseID)
	if err != nil {
		return store.Lease{}, err
	}
	if _, err := s.billing.GenerateRent(ctx, l, s.today()); err != nil {
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
	l, err := s.billNow(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	s.audit(r, "lease", l.ID, l.ID, "created", "Lease for "+l.TenantName+" at "+leasePlace(l)+" from "+l.StartDate.String()+" at "+money(l.RentCents)+" a month")
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
	l, err := s.billNow(r.Context(), userID(r), id)
	if err != nil {
		return err
	}
	s.audit(r, "lease", l.ID, l.ID, "updated", "Updated the lease of "+l.TenantName+" at "+leasePlace(l))
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
	s.audit(r, "lease", l.ID, l.ID, "terminated", "Lease of "+l.TenantName+" ended early on "+in.Date.String())
	writeJSON(w, http.StatusOK, s.withStatus(l))
	return nil
}

func (s *Server) deleteLease(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "lease")
	if err != nil {
		return err
	}
	l, err := s.store.GetLease(r.Context(), userID(r), id)
	if err != nil {
		return notFoundAs(err, "lease")
	}
	if err := s.store.DeleteLease(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "lease")
	}
	s.audit(r, "lease", id, id, "deleted", "Deleted the lease of "+l.TenantName+" at "+leasePlace(l))
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type ledgerCharge struct {
	store.Charge
	PaidCents      int64  `json:"paid_cents"`
	RemainingCents int64  `json:"remaining_cents"`
	Status         string `json:"status"`
}

type ledgerResponse struct {
	AsOf          domain.Date     `json:"as_of"`
	ChargedCents  int64           `json:"charged_cents"`
	PaidCents     int64           `json:"paid_cents"`
	BalanceCents  int64           `json:"balance_cents"`
	OverdueCents  int64           `json:"overdue_cents"`
	OldestOverdue *domain.Date    `json:"oldest_overdue"`
	DaysOverdue   int             `json:"days_overdue"`
	Charges       []ledgerCharge  `json:"charges"`
	Payments      []store.Payment `json:"payments"`
}

func (s *Server) leaseLedger(w http.ResponseWriter, r *http.Request) error {
	l, err := s.loadLease(r)
	if err != nil {
		return err
	}
	charges, err := s.store.ListCharges(r.Context(), userID(r), l.ID)
	if err != nil {
		return err
	}
	payments, _, err := s.store.ListPayments(r.Context(), userID(r), store.PaymentFilter{LeaseID: l.ID, Limit: 500})
	if err != nil {
		return err
	}

	in := make([]domain.LedgerCharge, len(charges))
	byID := make(map[string]store.Charge, len(charges))
	for i, c := range charges {
		in[i] = domain.LedgerCharge{ID: c.ID, Kind: c.Kind, DueDate: c.DueDate, AmountCents: c.AmountCents}
		byID[c.ID] = c
	}
	pays := make([]domain.LedgerPayment, len(payments))
	for i, p := range payments {
		pays[i] = domain.LedgerPayment{ID: p.ID, PaidOn: p.PaidOn, AmountCents: p.AmountCents}
	}

	asOf := s.today()
	ledger := domain.BuildLedger(in, pays, l.GraceDays, asOf)
	resp := ledgerResponse{
		AsOf:         asOf,
		ChargedCents: ledger.ChargedCents,
		PaidCents:    ledger.PaidCents,
		BalanceCents: ledger.BalanceCents,
		OverdueCents: ledger.OverdueCents,
		DaysOverdue:  ledger.DaysOverdue,
		Charges:      make([]ledgerCharge, 0, len(ledger.Charges)),
		Payments:     payments,
	}
	if !ledger.OldestOverdue.IsZero() {
		d := ledger.OldestOverdue
		resp.OldestOverdue = &d
	}
	for _, c := range ledger.Charges {
		resp.Charges = append(resp.Charges, ledgerCharge{
			Charge:         byID[c.ID],
			PaidCents:      c.PaidCents,
			RemainingCents: c.RemainingCents,
			Status:         c.Status,
		})
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

func (s *Server) createCharge(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "lease")
	if err != nil {
		return err
	}
	var in store.ChargeInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	trim(&in.Kind, &in.Description)

	var v validator
	v.check(domain.OneOf(in.Kind, domain.ManualChargeKinds), "kind",
		"must be one of "+strings.Join(domain.ManualChargeKinds, ", "))
	v.check(in.AmountCents > 0, "amount_cents", "must be more than zero")
	v.check(!in.DueDate.IsZero(), "due_date", "is required")
	v.maxLen(in.Description, "description", 200)
	if err := v.err(); err != nil {
		return err
	}

	chargeID, err := s.store.CreateCharge(r.Context(), userID(r), id, in)
	if err != nil {
		return notFoundAs(err, "lease")
	}
	s.audit(r, "charge", chargeID, id, "created", "Charged "+money(in.AmountCents)+" ("+in.Kind+") due "+in.DueDate.String())
	writeJSON(w, http.StatusCreated, map[string]string{"id": chargeID})
	return nil
}

func (s *Server) deleteCharge(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "charge")
	if err != nil {
		return err
	}
	if err := s.store.DeleteCharge(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "charge")
	}
	s.audit(r, "charge", id, "", "deleted", "Deleted a charge")
	w.WriteHeader(http.StatusNoContent)
	return nil
}
