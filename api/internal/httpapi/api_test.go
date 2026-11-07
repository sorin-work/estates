package httpapi

import (
	"net/http"
	"testing"

	"github.com/sorin-work/estates/api/internal/billing"
	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
	"github.com/sorin-work/estates/api/internal/testdb"
)

// These tests run the real router against Postgres. They are skipped when
// TEST_DATABASE_URL is not set.

func newDBServer(t *testing.T, today string) *client {
	t.Helper()
	st := store.New(testdb.New(t))
	srv := New(Options{
		Store:   st,
		Billing: billing.New(st, quietLogger()),
		Tokens:  testTokens(t),
		Logger:  quietLogger(),
		Today:   func() domain.Date { return domain.MustParseDate(today) },
	})
	c := &client{t: t, handler: srv.Handler()}
	var auth authResponse
	c.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
		"email": "owner@example.com", "name": "Owner", "password": "password123",
	}).decode(t, &auth)
	c.token = auth.Token
	return c
}

// newDBServerFor signs up another user on the same server as c.
func newDBServerFor(t *testing.T, c *client, email string) *client {
	t.Helper()
	other := &client{t: t, handler: c.handler}
	var auth authResponse
	other.mustStatus(http.StatusCreated, "POST", "/api/auth/register", map[string]string{
		"email": email, "name": "Someone Else", "password": "password123",
	}).decode(t, &auth)
	other.token = auth.Token
	return other
}

type idOnly struct {
	ID string `json:"id"`
}

func TestRentFlow(t *testing.T) {
	c := newDBServer(t, "2026-03-10")

	var prop, unit, tenant idOnly
	c.mustStatus(http.StatusCreated, "POST", "/api/properties", map[string]any{
		"name": "Cedar Flats", "kind": "multi_family", "address_line1": "8 Cedar Rd", "city": "Eugene",
	}).decode(t, &prop)
	c.mustStatus(http.StatusCreated, "POST", "/api/properties/"+prop.ID+"/units", map[string]any{
		"label": "2", "bedrooms": 1, "bathrooms": 1, "market_rent_cents": 110000,
	}).decode(t, &unit)
	c.mustStatus(http.StatusCreated, "POST", "/api/tenants", map[string]any{
		"first_name": "Grace", "last_name": "Hopper", "email": "grace@example.com",
	}).decode(t, &tenant)

	// Moved in mid January: prorated January, then full February and March.
	var lease store.Lease
	c.mustStatus(http.StatusCreated, "POST", "/api/leases", map[string]any{
		"unit_id": unit.ID, "tenant_id": tenant.ID, "start_date": "2026-01-16", "end_date": "2027-01-15",
		"rent_cents": 124000, "deposit_cents": 124000, "due_day": 1, "grace_days": 5, "late_fee_cents": 5000,
	}).decode(t, &lease)
	if lease.Status != "active" || lease.TenantName != "Grace Hopper" || lease.UnitLabel != "2" {
		t.Fatalf("lease %+v", lease)
	}

	// The same unit cannot be let twice.
	res := c.mustStatus(http.StatusConflict, "POST", "/api/leases", map[string]any{
		"unit_id": unit.ID, "tenant_id": tenant.ID, "start_date": "2026-06-01", "end_date": "2026-12-31", "rent_cents": 1,
	})
	if res.errorCode(t) != "conflict" {
		t.Errorf("overlap: %s", res.Body)
	}

	var ledger ledgerResponse
	c.mustStatus(http.StatusOK, "GET", "/api/leases/"+lease.ID+"/ledger", nil).decode(t, &ledger)
	if len(ledger.Charges) != 3 {
		t.Fatalf("charges %+v", ledger.Charges)
	}
	// 16 of 31 days of January.
	if ledger.Charges[0].AmountCents != 64000 || ledger.ChargedCents != 64000+124000+124000 {
		t.Errorf("charged %d, first %d", ledger.ChargedCents, ledger.Charges[0].AmountCents)
	}

	// Pay January and February. March is due but within the grace period
	// until the 6th, so on the 10th it is overdue.
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+lease.ID+"/payments", map[string]any{
		"amount_cents": 188000, "paid_on": "2026-02-02", "method": "bank_transfer",
	})
	res = c.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/leases/"+lease.ID+"/payments", map[string]any{
		"amount_cents": 100, "paid_on": "2026-04-01",
	})
	if res.errorFields(t)["paid_on"] == "" {
		t.Errorf("future payment: %s", res.Body)
	}

	var balances []billing.Balance
	c.mustStatus(http.StatusOK, "GET", "/api/rent/balances?overdue=true", nil).decode(t, &balances)
	if len(balances) != 1 || balances[0].OverdueCents != 124000 || balances[0].DaysOverdue != 9 {
		t.Fatalf("balances %+v", balances)
	}

	// The billing run adds a late fee for March, once.
	var run billing.Result
	c.mustStatus(http.StatusOK, "POST", "/api/rent/run", nil).decode(t, &run)
	if run.LateFees != 1 {
		t.Errorf("first run %+v", run)
	}
	c.mustStatus(http.StatusOK, "POST", "/api/rent/run", nil).decode(t, &run)
	if run.LateFees != 0 || run.Rent != 0 {
		t.Errorf("second run should add nothing: %+v", run)
	}

	c.mustStatus(http.StatusOK, "GET", "/api/leases/"+lease.ID+"/ledger", nil).decode(t, &ledger)
	if ledger.BalanceCents != 124000+5000 {
		t.Errorf("balance %d", ledger.BalanceCents)
	}
	last := ledger.Charges[len(ledger.Charges)-1]
	if last.Kind != "late_fee" || last.DueDate.String() != "2026-03-07" {
		t.Errorf("late fee %+v", last)
	}

	var page paymentList
	c.mustStatus(http.StatusOK, "GET", "/api/payments?from=2026-02-01&to=2026-02-28", nil).decode(t, &page)
	if page.Total != 1 || page.TotalCents != 188000 {
		t.Errorf("payments %+v", page)
	}

	// A lease with history cannot disappear with its property.
	c.mustStatus(http.StatusConflict, "DELETE", "/api/properties/"+prop.ID, nil)

	var dash dashboardResponse
	c.mustStatus(http.StatusOK, "GET", "/api/dashboard", nil).decode(t, &dash)
	if dash.Occupancy.Units != 1 || dash.Occupancy.Occupied != 1 || dash.Occupancy.Rate != 1 {
		t.Errorf("occupancy %+v", dash.Occupancy)
	}
	// The late fee is owed but has its own grace period, so it is not
	// overdue yet.
	if dash.Outstanding.OverdueLeases != 1 || dash.Outstanding.OverdueCents != 124000 || dash.Outstanding.BalanceCents != 129000 {
		t.Errorf("outstanding %+v", dash.Outstanding)
	}
	if len(dash.Monthly) != 12 || dash.ThisMonth.Month != "2026-03" || dash.ThisMonth.ChargedCents != 129000 {
		t.Errorf("this month %+v (%d months)", dash.ThisMonth, len(dash.Monthly))
	}

	// A lease cannot be ended early after it has already ended.
	res = c.mustStatus(http.StatusConflict, "POST", "/api/leases/"+lease.ID+"/terminate", map[string]string{"date": "2027-02-01"})
	if res.errorCode(t) != "conflict" {
		t.Errorf("terminate after end: %s", res.Body)
	}

	// Ending the lease early drops rent due after the termination date.
	c.mustStatus(http.StatusOK, "POST", "/api/leases/"+lease.ID+"/terminate", map[string]string{"date": "2026-03-31"})
	var leases []store.Lease
	c.mustStatus(http.StatusOK, "GET", "/api/leases?status=active", nil).decode(t, &leases)
	if len(leases) != 1 || leases[0].TerminatedOn == nil {
		t.Errorf("leases %+v", leases)
	}
}

func TestMaintenanceAndExpensesFlow(t *testing.T) {
	c := newDBServer(t, "2026-05-01")

	var prop idOnly
	c.mustStatus(http.StatusCreated, "POST", "/api/properties", map[string]any{
		"name": "Pine Cottage", "address_line1": "3 Pine Way", "city": "Hood River",
	}).decode(t, &prop)

	var req maintenanceView
	c.mustStatus(http.StatusCreated, "POST", "/api/maintenance", map[string]any{
		"property_id": prop.ID, "title": "Roof leak over porch", "priority": "urgent",
	}).decode(t, &req)
	if req.Status != "open" || req.ReportedOn.String() != "2026-05-01" || len(req.NextStatuses) == 0 {
		t.Fatalf("request %+v", req)
	}

	c.mustStatus(http.StatusOK, "POST", "/api/maintenance/"+req.ID+"/status", map[string]string{"status": "in_progress"})
	c.mustStatus(http.StatusOK, "POST", "/api/maintenance/"+req.ID+"/status", map[string]string{"status": "completed"}).decode(t, &req)
	if req.CompletedOn == nil {
		t.Error("completed request has no completion date")
	}
	c.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/maintenance/"+req.ID+"/status", map[string]string{"status": "on_hold"})

	c.mustStatus(http.StatusCreated, "POST", "/api/expenses", map[string]any{
		"property_id": prop.ID, "maintenance_request_id": req.ID, "category": "repairs",
		"amount_cents": 65000, "spent_on": "2026-04-30", "vendor": "Summit Roofing",
	})
	c.mustStatus(http.StatusCreated, "POST", "/api/expenses", map[string]any{
		"property_id": prop.ID, "category": "insurance", "amount_cents": 90000, "spent_on": "2026-04-02",
	})
	res := c.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/expenses", map[string]any{
		"property_id": prop.ID, "category": "snacks", "amount_cents": 0,
	})
	if f := res.errorFields(t); f["category"] == "" || f["amount_cents"] == "" || f["spent_on"] == "" {
		t.Errorf("fields %v", f)
	}

	var list expenseList
	c.mustStatus(http.StatusOK, "GET", "/api/expenses?property_id="+prop.ID+"&from=2026-04-01&to=2026-04-30", nil).decode(t, &list)
	if len(list.Expenses) != 2 || list.TotalCents != 155000 || list.ByCategory[0].Category != "insurance" {
		t.Errorf("expenses %+v", list)
	}

	c.mustStatus(http.StatusOK, "GET", "/api/maintenance/"+req.ID, nil).decode(t, &req)
	if req.CostCents != 65000 {
		t.Errorf("cost %d", req.CostCents)
	}

	var active []maintenanceView
	c.mustStatus(http.StatusOK, "GET", "/api/maintenance?status=active", nil).decode(t, &active)
	if len(active) != 0 {
		t.Errorf("active %+v", active)
	}
	c.mustStatus(http.StatusBadRequest, "GET", "/api/maintenance?status=whenever", nil)

	var found []maintenanceView
	c.mustStatus(http.StatusOK, "GET", "/api/maintenance?q=ROOF", nil).decode(t, &found)
	if len(found) != 1 || found[0].ID != req.ID {
		t.Errorf("search %+v", found)
	}
	c.mustStatus(http.StatusOK, "GET", "/api/maintenance?q=plumbing", nil).decode(t, &found)
	if len(found) != 0 {
		t.Errorf("search for something else found %d", len(found))
	}
}
