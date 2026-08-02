package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestProfitLoss(t *testing.T) {
	c := newDBServer(t, "2026-03-15")
	l := newLease(t, c, map[string]any{"rent_cents": 150000})

	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/payments", map[string]any{"amount_cents": 150000, "paid_on": "2026-01-02"})
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/payments", map[string]any{"amount_cents": 150000, "paid_on": "2026-02-01"})
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/deposit", map[string]any{
		"kind": "received", "amount_cents": 150000, "occurred_on": "2025-12-20",
	})
	c.mustStatus(http.StatusCreated, "POST", "/api/expenses", map[string]any{
		"property_id": l.PropertyID, "category": "insurance", "amount_cents": 60000, "spent_on": "2026-01-10",
	})
	c.mustStatus(http.StatusCreated, "POST", "/api/expenses", map[string]any{
		"property_id": l.PropertyID, "category": "repairs", "amount_cents": 12500, "spent_on": "2026-02-20",
	})
	// Outside the period.
	c.mustStatus(http.StatusCreated, "POST", "/api/expenses", map[string]any{
		"property_id": l.PropertyID, "category": "repairs", "amount_cents": 99900, "spent_on": "2025-12-30",
	})

	var pl profitLossResponse
	c.mustStatus(http.StatusOK, "GET", "/api/reports/profit-loss?from=2026-01-01&to=2026-02-28", nil).decode(t, &pl)
	if len(pl.Properties) != 1 {
		t.Fatalf("properties %+v", pl.Properties)
	}
	p := pl.Properties[0]
	// The deposit is not income.
	if p.ChargedCents != 300000 || p.CollectedCents != 300000 || p.ExpensesCents != 72500 || p.NetCents != 227500 {
		t.Errorf("result %+v", p)
	}
	if len(p.ByCategory) != 2 || p.ByCategory[0].Category != "insurance" {
		t.Errorf("by category %+v", p.ByCategory)
	}
	if pl.Totals.NetCents != 227500 || len(pl.Totals.ByCategory) != 2 {
		t.Errorf("totals %+v", pl.Totals)
	}

	// The year to date by default.
	c.mustStatus(http.StatusOK, "GET", "/api/reports/profit-loss", nil).decode(t, &pl)
	if pl.From.String() != "2026-01-01" || pl.To.String() != "2026-03-15" {
		t.Errorf("default period %s to %s", pl.From, pl.To)
	}
	c.mustStatus(http.StatusBadRequest, "GET", "/api/reports/profit-loss?from=2026-03-01&to=2026-01-01", nil)

	res := c.mustStatus(http.StatusOK, "GET", "/api/reports/profit-loss.csv?from=2026-01-01&to=2026-02-28", nil)
	lines := strings.Split(strings.TrimSpace(string(res.Body)), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "Property,") ||
		!strings.HasPrefix(lines[1], "Alder House,3000.00,3000.00,0.00,725.00,125.00,") ||
		!strings.HasSuffix(lines[2], ",2275.00") {
		t.Errorf("csv:\n%s", res.Body)
	}
}

func TestPaymentAndExpenseExports(t *testing.T) {
	c := newDBServer(t, "2026-03-15")
	l := newLease(t, c, nil)
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/payments", map[string]any{
		"amount_cents": 150000, "paid_on": "2026-02-01", "method": "check", "reference": "#1001",
	})
	c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+l.ID+"/payments", map[string]any{
		"amount_cents": 150000, "paid_on": "2026-03-01",
	})
	c.mustStatus(http.StatusCreated, "POST", "/api/expenses", map[string]any{
		"property_id": l.PropertyID, "category": "repairs", "amount_cents": 12500, "spent_on": "2026-02-20",
		"vendor": "Hardware Depot", "description": "=cmd|' /C calc'!A0",
	})

	res := c.mustStatus(http.StatusOK, "GET", "/api/payments/export.csv?from=2026-02-01&to=2026-02-28", nil)
	want := "Date,Tenant,Property,Unit,Amount,Method,Reference,Notes\n2026-02-01,Rosa Parks,Alder House,Main,1500.00,check,#1001,\n"
	if string(res.Body) != want {
		t.Errorf("payments csv:\n%s", res.Body)
	}

	res = c.mustStatus(http.StatusOK, "GET", "/api/expenses/export.csv", nil)
	lines := strings.Split(strings.TrimSpace(string(res.Body)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[1], `,'=cmd|' /C calc'!A0,`) || !strings.HasSuffix(lines[1], ",125.00") {
		t.Errorf("expenses csv:\n%s", res.Body)
	}
}
