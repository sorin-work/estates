package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/sorin-work/estates/api/internal/billing"
	"github.com/sorin-work/estates/api/internal/store"
	"github.com/sorin-work/estates/api/internal/testdb"
)

// At 23:30 UTC on 31 March it is already 1 April in Auckland, so a lease
// starting on 1 April has begun for a landlord there and not yet for one in
// UTC.
func TestTodayFollowsTheUsersTimeZone(t *testing.T) {
	st := store.New(testdb.New(t))
	now := time.Date(2026, 3, 31, 23, 30, 0, 0, time.UTC)
	srv := New(Options{
		Store:   st,
		Billing: billing.New(st, quietLogger()),
		Tokens:  testTokens(t),
		Logger:  quietLogger(),
		Now:     func() time.Time { return now },
	})
	handler := srv.Handler()

	status := func(email, zone string) string {
		c := &client{t: t, handler: handler}
		var auth authResponse
		c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
			"email": email, "name": "Owner", "password": "password123",
		}).decode(t, &auth)
		c.token = auth.Token
		var me struct {
			Today string `json:"today"`
		}
		c.mustStatus(http.StatusOK, "PUT", "/api/auth/me", map[string]any{
			"name": "Owner", "currency": "USD", "deposit_return_days": 30, "time_zone": zone,
		}).decode(t, &me)
		wantToday := "2026-03-31"
		if zone == "Pacific/Auckland" {
			wantToday = "2026-04-01"
		}
		if me.Today != wantToday {
			t.Errorf("%s: today = %q, want %s", zone, me.Today, wantToday)
		}
		var prop, unit, tenant, lease idOnly
		c.mustStatus(http.StatusCreated, "POST", "/api/properties", map[string]any{
			"name": "Kauri", "kind": "condo", "address_line1": "1 Queen St", "city": "Auckland",
		}).decode(t, &prop)
		c.mustStatus(http.StatusCreated, "POST", "/api/properties/"+prop.ID+"/units", map[string]any{"label": "1", "bedrooms": 1, "bathrooms": 1}).decode(t, &unit)
		c.mustStatus(http.StatusCreated, "POST", "/api/tenants", map[string]any{"first_name": "Mere", "last_name": "Ngata"}).decode(t, &tenant)
		c.mustStatus(http.StatusCreated, "POST", "/api/leases", map[string]any{
			"unit_id": unit.ID, "tenant_id": tenant.ID, "start_date": "2026-04-01", "end_date": "2027-03-31",
			"rent_cents": 200000, "due_day": 1,
		}).decode(t, &lease)

		var got struct {
			Status string `json:"status"`
		}
		c.mustStatus(http.StatusOK, "GET", "/api/leases/"+lease.ID, nil).decode(t, &got)

		var ledger struct {
			AsOf    string            `json:"as_of"`
			Charges []json.RawMessage `json:"charges"`
		}
		c.mustStatus(http.StatusOK, "GET", "/api/leases/"+lease.ID+"/ledger", nil).decode(t, &ledger)
		wantAsOf, wantCharges := "2026-03-31", 0
		if zone == "Pacific/Auckland" {
			// The first rent was charged when the lease was created.
			wantAsOf, wantCharges = "2026-04-01", 1
		}
		if ledger.AsOf != wantAsOf || len(ledger.Charges) != wantCharges {
			t.Errorf("%s: ledger as of %s with %d charges, want %s with %d", zone, ledger.AsOf, len(ledger.Charges), wantAsOf, wantCharges)
		}
		return got.Status
	}

	if got := status("auckland@example.com", "Pacific/Auckland"); got != "active" {
		t.Errorf("Auckland: lease is %s, want active", got)
	}
	if got := status("utc@example.com", "UTC"); got != "upcoming" {
		t.Errorf("UTC: lease is %s, want upcoming", got)
	}
}
