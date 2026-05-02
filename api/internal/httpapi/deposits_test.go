package httpapi

import (
	"net/http"
	"testing"

	"github.com/sorin-work/estates/api/internal/store"
)

// newLease creates a property, unit and tenant and a lease on them. Fields
// in terms override the defaults.
func newLease(t *testing.T, c *client, terms map[string]any) store.Lease {
	t.Helper()
	var prop, unit, tenant idOnly
	c.mustStatus(http.StatusCreated, "POST", "/api/properties", map[string]any{
		"name": "Alder House", "address_line1": "5 Alder St", "city": "Corvallis",
	}).decode(t, &prop)
	c.mustStatus(http.StatusCreated, "POST", "/api/properties/"+prop.ID+"/units", map[string]any{
		"label": "Main", "bedrooms": 2, "bathrooms": 1,
	}).decode(t, &unit)
	c.mustStatus(http.StatusCreated, "POST", "/api/tenants", map[string]any{
		"first_name": "Rosa", "last_name": "Parks",
	}).decode(t, &tenant)

	body := map[string]any{
		"unit_id": unit.ID, "tenant_id": tenant.ID, "start_date": "2026-01-01", "end_date": "2026-12-31",
		"rent_cents": 150000, "deposit_cents": 150000, "due_day": 1, "grace_days": 5,
	}
	for k, v := range terms {
		body[k] = v
	}
	var l store.Lease
	c.mustStatus(http.StatusCreated, "POST", "/api/leases", body).decode(t, &l)
	return l
}

func TestDepositFlow(t *testing.T) {
	c := newDBServer(t, "2026-03-10")
	l := newLease(t, c, nil)
	path := "/api/leases/" + l.ID + "/deposit"

	var dep depositResponse
	c.mustStatus(http.StatusOK, "GET", path, nil).decode(t, &dep)
	if dep.AgreedCents != 150000 || dep.HeldCents != 0 || len(dep.Transactions) != 0 {
		t.Fatalf("new lease deposit %+v", dep)
	}

	c.mustStatus(http.StatusCreated, "POST", path, map[string]any{
		"kind": "received", "amount_cents": 150000, "occurred_on": "2025-12-28", "notes": "Check #1042",
	}).decode(t, &dep)
	if dep.HeldCents != 150000 || len(dep.Transactions) != 1 {
		t.Fatalf("after receiving %+v", dep)
	}

	// Paying out more than is held is refused.
	res := c.mustStatus(http.StatusConflict, "POST", path, map[string]any{
		"kind": "returned", "amount_cents": 150001, "occurred_on": "2026-03-01",
	})
	if res.errorCode(t) != "conflict" {
		t.Errorf("over-return: %s", res.Body)
	}
	res = c.mustStatus(http.StatusUnprocessableEntity, "POST", path, map[string]any{
		"kind": "lost", "amount_cents": 0, "occurred_on": "2026-04-01",
	})
	if f := res.errorFields(t); f["kind"] == "" || f["amount_cents"] == "" || f["occurred_on"] == "" {
		t.Errorf("fields %v", f)
	}

	c.mustStatus(http.StatusCreated, "POST", path, map[string]any{
		"kind": "withheld", "amount_cents": 40000, "occurred_on": "2026-03-05", "notes": "Carpet cleaning",
	})
	c.mustStatus(http.StatusCreated, "POST", path, map[string]any{
		"kind": "returned", "amount_cents": 110000, "occurred_on": "2026-03-05",
	}).decode(t, &dep)
	if dep.HeldCents != 0 || dep.WithheldCents != 40000 || dep.ReturnedCents != 110000 {
		t.Fatalf("after return %+v", dep)
	}

	// The received deposit has been paid out, so it cannot be deleted, but
	// the return can.
	c.mustStatus(http.StatusConflict, "DELETE", "/api/deposit-transactions/"+dep.Transactions[0].ID, nil)
	c.mustStatus(http.StatusNoContent, "DELETE", "/api/deposit-transactions/"+dep.Transactions[2].ID, nil)
	c.mustStatus(http.StatusOK, "GET", path, nil).decode(t, &dep)
	if dep.HeldCents != 110000 {
		t.Errorf("held after deleting the return: %d", dep.HeldCents)
	}

	// Deposits are not rent: the ledger is untouched.
	var ledger ledgerResponse
	c.mustStatus(http.StatusOK, "GET", "/api/leases/"+l.ID+"/ledger", nil).decode(t, &ledger)
	if ledger.PaidCents != 0 {
		t.Errorf("deposit counted as a payment: %+v", ledger)
	}
}

func TestDepositsAreNotRentCollected(t *testing.T) {
	c := newDBServer(t, "2026-03-10")
	l := newLease(t, c, nil)

	// Deposits can no longer be added as a charge.
	res := c.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/leases/"+l.ID+"/charges", map[string]any{
		"kind": "deposit", "amount_cents": 150000, "due_date": "2026-01-01",
	})
	if res.errorFields(t)["kind"] == "" {
		t.Errorf("deposit charge: %s", res.Body)
	}

	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/deposit", map[string]any{
		"kind": "received", "amount_cents": 150000, "occurred_on": "2026-03-02",
	})
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/payments", map[string]any{
		"amount_cents": 150000, "paid_on": "2026-03-02",
	})
	var dash dashboardResponse
	c.mustStatus(http.StatusOK, "GET", "/api/dashboard", nil).decode(t, &dash)
	if dash.ThisMonth.CollectedCents != 150000 {
		t.Errorf("collected %d, want only the rent payment", dash.ThisMonth.CollectedCents)
	}
}
