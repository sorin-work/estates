package httpapi

import (
	"net/http"
	"testing"

	"github.com/sorin-work/estates/api/internal/store"
)

func TestVendorFlow(t *testing.T) {
	c := newDBServer(t, "2026-07-01")

	var prop idOnly
	c.mustStatus(http.StatusCreated, "POST", "/api/properties", map[string]any{
		"name": "Fir Court", "address_line1": "9 Fir Ct", "city": "Bend",
	}).decode(t, &prop)

	var plumber store.Vendor
	c.mustStatus(http.StatusCreated, "POST", "/api/vendors", map[string]any{
		"name": "Rapid Rooter", "trade": "plumbing", "phone": "(541) 555-0100", "email": "Jobs@RapidRooter.example",
	}).decode(t, &plumber)
	if plumber.Email != "jobs@rapidrooter.example" || plumber.SpentCents != 0 {
		t.Fatalf("vendor %+v", plumber)
	}
	c.mustStatus(http.StatusConflict, "POST", "/api/vendors", map[string]any{"name": "rapid rooter"})
	res := c.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/vendors", map[string]any{"name": "", "trade": "wizardry"})
	if f := res.errorFields(t); f["name"] == "" || f["trade"] == "" {
		t.Errorf("fields %v", f)
	}

	var job maintenanceView
	c.mustStatus(http.StatusCreated, "POST", "/api/maintenance", map[string]any{
		"property_id": prop.ID, "title": "Clogged drain", "vendor_id": plumber.ID,
	}).decode(t, &job)
	if job.Vendor != "Rapid Rooter" || job.VendorID == nil || *job.VendorID != plumber.ID {
		t.Fatalf("job %+v", job)
	}
	c.mustStatus(http.StatusCreated, "POST", "/api/expenses", map[string]any{
		"property_id": prop.ID, "maintenance_request_id": job.ID, "vendor_id": plumber.ID,
		"category": "repairs", "amount_cents": 18500, "spent_on": "2026-06-28",
	})

	var got store.Vendor
	c.mustStatus(http.StatusOK, "GET", "/api/vendors/"+plumber.ID, nil).decode(t, &got)
	if got.OpenJobs != 1 || got.SpentCents != 18500 || got.LastUsedOn == nil || got.LastUsedOn.String() != "2026-06-28" {
		t.Errorf("vendor totals %+v", got)
	}

	var list expenseList
	c.mustStatus(http.StatusOK, "GET", "/api/expenses?vendor_id="+plumber.ID, nil).decode(t, &list)
	if len(list.Expenses) != 1 || list.Expenses[0].Vendor != "Rapid Rooter" {
		t.Errorf("expenses by vendor %+v", list.Expenses)
	}

	// Another owner's vendor cannot be linked.
	other := newDBServerFor(t, c, "other@example.com")
	var theirs store.Vendor
	other.mustStatus(http.StatusCreated, "POST", "/api/vendors", map[string]any{"name": "Elsewhere Electric"}).decode(t, &theirs)
	c.mustStatus(http.StatusConflict, "POST", "/api/maintenance", map[string]any{
		"property_id": prop.ID, "title": "Flickering lights", "vendor_id": theirs.ID,
	})

	// Deleting the vendor keeps the name on the job.
	c.mustStatus(http.StatusNoContent, "DELETE", "/api/vendors/"+plumber.ID, nil)
	c.mustStatus(http.StatusOK, "GET", "/api/maintenance/"+job.ID, nil).decode(t, &job)
	if job.VendorID != nil || job.Vendor != "Rapid Rooter" {
		t.Errorf("job after deleting vendor %+v", job)
	}
}
