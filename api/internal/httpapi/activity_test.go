package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/sorin-work/estates/api/internal/store"
)

func TestMoney(t *testing.T) {
	for cents, want := range map[int64]string{5: "0.05", 145000: "1,450.00", 123456789: "1,234,567.89", -250000: "-2,500.00"} {
		if got := money(cents); got != want {
			t.Errorf("money(%d) = %s, want %s", cents, got, want)
		}
	}
}

func TestActivityHistory(t *testing.T) {
	c := newDBServer(t, "2026-03-15")
	l := newLease(t, c, map[string]any{"rent_cents": 145000})
	var p store.Payment
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/payments", map[string]any{
		"amount_cents": 145000, "paid_on": "2026-03-01",
	}).decode(t, &p)
	c.mustStatus(http.StatusNoContent, "DELETE", "/api/payments/"+p.ID, nil)

	var history []store.Activity
	c.mustStatus(http.StatusOK, "GET", "/api/activity?lease_id="+l.ID, nil).decode(t, &history)
	var got []string
	for _, a := range history {
		got = append(got, a.EntityType+" "+a.Action)
	}
	if strings.Join(got, ", ") != "payment deleted, payment created, lease created" {
		t.Fatalf("lease history %v", got)
	}
	if history[1].Summary != "Payment of 1,450.00 from Rosa Parks on 2026-03-01" {
		t.Errorf("summary %q", history[1].Summary)
	}

	var all []store.Activity
	c.mustStatus(http.StatusOK, "GET", "/api/activity?limit=2", nil).decode(t, &all)
	if len(all) != 2 || all[0].ID <= all[1].ID {
		t.Fatalf("newest first: %+v", all)
	}
	var older []store.Activity
	c.mustStatus(http.StatusOK, "GET", "/api/activity?before="+itoa(all[1].ID), nil).decode(t, &older)
	// Property and tenant created, then the lease.
	if len(older) != 3 || older[0].EntityType != "lease" {
		t.Errorf("older %+v", older)
	}

	other := newDBServerFor(t, c, "other@example.com")
	var theirs []store.Activity
	other.mustStatus(http.StatusOK, "GET", "/api/activity", nil).decode(t, &theirs)
	if len(theirs) != 0 {
		t.Errorf("other owner sees %d entries", len(theirs))
	}
}
