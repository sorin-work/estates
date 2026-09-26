package httpapi

import (
	"net/http"
	"testing"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

func TestRentIncrease(t *testing.T) {
	c := newDBServer(t, "2026-05-10")
	l := newLease(t, c, map[string]any{"rent_cents": 100000})

	res := c.mustStatus(http.StatusConflict, "POST", "/api/leases/"+l.ID+"/rent-changes", map[string]any{
		"effective_date": "2027-02-01", "rent_cents": 105000,
	})
	if res.errorCode(t) != "conflict" {
		t.Errorf("after the end: %s", res.Body)
	}

	// Backdated to April: April and May are re-charged at the new rent.
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/rent-changes", map[string]any{
		"effective_date": "2026-04-01", "rent_cents": 104000, "notes": "Annual increase, 60 days notice",
	}).decode(t, &l)
	if l.CurrentRentCents != 104000 || len(l.RentChanges) != 1 {
		t.Fatalf("lease %+v", l)
	}
	c.mustStatus(http.StatusConflict, "POST", "/api/leases/"+l.ID+"/rent-changes", map[string]any{
		"effective_date": "2026-04-01", "rent_cents": 106000,
	})

	var ledger ledgerResponse
	c.mustStatus(http.StatusOK, "GET", "/api/leases/"+l.ID+"/ledger", nil).decode(t, &ledger)
	if ledger.ChargedCents != 3*100000+2*104000 {
		t.Errorf("charged %d", ledger.ChargedCents)
	}

	c.mustStatus(http.StatusNoContent, "DELETE", "/api/rent-changes/"+l.RentChanges[0].ID, nil)
	c.mustStatus(http.StatusOK, "GET", "/api/leases/"+l.ID+"/ledger", nil).decode(t, &ledger)
	if ledger.ChargedCents != 5*100000 {
		t.Errorf("charged after removing the increase %d", ledger.ChargedCents)
	}
}

func TestRenewLease(t *testing.T) {
	c := newDBServer(t, "2026-11-20")
	l := newLease(t, c, map[string]any{"rent_cents": 150000, "deposit_cents": 150000})
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/deposit", map[string]any{
		"kind": "received", "amount_cents": 150000, "occurred_on": "2025-12-28",
	})

	// It shows as ending soon until it is renewed.
	var dash dashboardResponse
	c.mustStatus(http.StatusOK, "GET", "/api/dashboard", nil).decode(t, &dash)
	if len(dash.ExpiringLeases) != 1 {
		t.Fatalf("expiring %+v", dash.ExpiringLeases)
	}

	res := c.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/leases/"+l.ID+"/renew", map[string]any{})
	if f := res.errorFields(t); f["end_date"] == "" || f["rent_cents"] == "" {
		t.Errorf("fields %v", f)
	}

	var renewal store.Lease
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/renew", map[string]any{
		"end_date": "2027-12-31", "rent_cents": 156000,
	}).decode(t, &renewal)
	if renewal.StartDate.String() != "2027-01-01" || renewal.RentCents != 156000 || renewal.Status != "upcoming" ||
		renewal.RenewedFromID == nil || *renewal.RenewedFromID != l.ID || renewal.UnitID != l.UnitID {
		t.Fatalf("renewal %+v", renewal)
	}
	c.mustStatus(http.StatusConflict, "POST", "/api/leases/"+l.ID+"/renew", map[string]any{
		"end_date": "2027-12-31", "rent_cents": 156000,
	})

	c.mustStatus(http.StatusOK, "GET", "/api/leases/"+l.ID, nil).decode(t, &l)
	if l.RenewedByID == nil || *l.RenewedByID != renewal.ID {
		t.Errorf("old lease %+v", l)
	}

	// The deposit moved with the tenant.
	var dep depositResponse
	c.mustStatus(http.StatusOK, "GET", "/api/leases/"+renewal.ID+"/deposit", nil).decode(t, &dep)
	if dep.HeldCents != 150000 {
		t.Errorf("deposit on the renewal %+v", dep)
	}

	c.mustStatus(http.StatusOK, "GET", "/api/dashboard", nil).decode(t, &dash)
	if len(dash.ExpiringLeases) != 0 {
		t.Errorf("renewed lease still expiring: %+v", dash.ExpiringLeases)
	}
}

func TestReminders(t *testing.T) {
	c := newDBServer(t, "2026-11-20")
	l := newLease(t, c, nil)

	var got []domain.Reminder
	c.mustStatus(http.StatusOK, "GET", "/api/reminders", nil).decode(t, &got)
	if len(got) != 1 || got[0].Kind != domain.ReminderLeaseEnding || got[0].LeaseID != l.ID || got[0].Title != "Lease ends in 41 days" {
		t.Fatalf("reminders %+v", got)
	}

	// Once renewed there is nothing to do.
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/renew", map[string]any{
		"end_date": "2027-12-31", "rent_cents": 150000,
	})
	c.mustStatus(http.StatusOK, "GET", "/api/reminders", nil).decode(t, &got)
	if len(got) != 0 {
		t.Errorf("after renewal %+v", got)
	}
}
