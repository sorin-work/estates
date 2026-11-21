package httpapi

import (
	"net/http"
	"testing"
)

func TestLeaseStatement(t *testing.T) {
	c := newDBServer(t, "2026-03-10")

	var prop, unit, tenant, lease idOnly
	c.mustStatus(http.StatusCreated, "POST", "/api/properties", map[string]any{
		"name": "Cedar Flats", "kind": "multi_family", "address_line1": "8 Cedar Rd", "city": "Eugene",
	}).decode(t, &prop)
	c.mustStatus(http.StatusCreated, "POST", "/api/properties/"+prop.ID+"/units", map[string]any{
		"label": "2", "bedrooms": 1, "bathrooms": 1,
	}).decode(t, &unit)
	c.mustStatus(http.StatusCreated, "POST", "/api/tenants", map[string]any{
		"first_name": "Grace", "last_name": "Hopper", "email": "grace@example.com",
	}).decode(t, &tenant)
	c.mustStatus(http.StatusCreated, "POST", "/api/leases", map[string]any{
		"unit_id": unit.ID, "tenant_id": tenant.ID, "start_date": "2026-01-01", "end_date": "2026-12-31",
		"rent_cents": 100000, "due_day": 1, "grace_days": 5,
	}).decode(t, &lease)
	for _, p := range []map[string]any{
		{"amount_cents": 100000, "paid_on": "2026-01-01", "method": "bank_transfer"},
		{"amount_cents": 50000, "paid_on": "2026-02-04", "method": "check", "reference": "#104"},
	} {
		c.mustStatus(http.StatusCreated, "POST", "/api/leases/"+lease.ID+"/payments", p)
	}

	type line struct {
		ID           string `json:"id"`
		Date         string `json:"date"`
		Payment      bool   `json:"payment"`
		BalanceCents int64  `json:"balance_cents"`
	}
	var st struct {
		From         string `json:"from"`
		To           string `json:"to"`
		OpeningCents int64  `json:"opening_cents"`
		ClosingCents int64  `json:"closing_cents"`
		Lines        []line `json:"lines"`
		Tenant       struct {
			Name string `json:"name"`
		} `json:"tenant"`
		Property struct {
			Name      string `json:"name"`
			UnitLabel string `json:"unit_label"`
		} `json:"property"`
	}

	// By default: from the lease start to today. January to March rent,
	// two payments.
	c.mustStatus(http.StatusOK, "GET", "/api/leases/"+lease.ID+"/statement", nil).decode(t, &st)
	if st.From != "2026-01-01" || st.To != "2026-03-10" || st.OpeningCents != 0 || st.ClosingCents != 150000 {
		t.Errorf("statement %+v", st)
	}
	if len(st.Lines) != 5 || st.Lines[0].Payment || !st.Lines[1].Payment || st.Lines[1].BalanceCents != 0 {
		t.Errorf("lines %+v", st.Lines)
	}
	if st.Tenant.Name != "Grace Hopper" || st.Property.Name != "Cedar Flats" || st.Property.UnitLabel != "2" {
		t.Errorf("letterhead %+v %+v", st.Tenant, st.Property)
	}

	c.mustStatus(http.StatusOK, "GET", "/api/leases/"+lease.ID+"/statement?from=2026-02-01&to=2026-02-28", nil).decode(t, &st)
	if st.OpeningCents != 0 || st.ClosingCents != 50000 || len(st.Lines) != 2 {
		t.Errorf("February %+v", st)
	}

	c.mustStatus(http.StatusBadRequest, "GET", "/api/leases/"+lease.ID+"/statement?from=2026-03-01&to=2026-02-01", nil)
	c.mustStatus(http.StatusBadRequest, "GET", "/api/leases/"+lease.ID+"/statement?from=March", nil)

	other := newDBServerFor(t, c, "other@example.com")
	other.mustStatus(http.StatusNotFound, "GET", "/api/leases/"+lease.ID+"/statement", nil)
}
