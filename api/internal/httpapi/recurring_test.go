package httpapi

import (
	"net/http"
	"testing"
)

func TestRecurringExpenseFlow(t *testing.T) {
	c := newDBServer(t, "2026-07-20")
	var prop idOnly
	c.mustStatus(http.StatusCreated, "POST", "/api/properties", map[string]any{
		"name": "Oak Row", "address_line1": "2 Oak Row", "city": "Salem",
	}).decode(t, &prop)

	res := c.mustStatus(http.StatusUnprocessableEntity, "POST", "/api/recurring-expenses", map[string]any{
		"property_id": prop.ID, "category": "mortgage", "amount_cents": 1, "day_of_month": 32, "every_months": 2,
	})
	if f := res.errorFields(t); f["day_of_month"] == "" || f["every_months"] == "" || f["start_date"] == "" {
		t.Errorf("fields %v", f)
	}

	// Started in May: May, June and July have been created already.
	var re recurringView
	c.mustStatus(http.StatusCreated, "POST", "/api/recurring-expenses", map[string]any{
		"property_id": prop.ID, "vendor": "First Coastal Bank", "category": "mortgage", "amount_cents": 142000,
		"description": "Mortgage payment", "day_of_month": 1, "start_date": "2026-05-01",
	}).decode(t, &re)
	if re.EveryMonths != 1 || re.GeneratedThrough == nil || re.GeneratedThrough.String() != "2026-07-01" {
		t.Fatalf("recurring %+v", re)
	}
	if re.NextOn == nil || re.NextOn.String() != "2026-08-01" {
		t.Errorf("next on %v", re.NextOn)
	}

	var list expenseList
	c.mustStatus(http.StatusOK, "GET", "/api/expenses?property_id="+prop.ID, nil).decode(t, &list)
	if len(list.Expenses) != 3 || list.TotalCents != 3*142000 {
		t.Errorf("expenses %+v", list)
	}

	// Quarterly insurance that has not come round yet.
	c.mustStatus(http.StatusCreated, "POST", "/api/recurring-expenses", map[string]any{
		"property_id": prop.ID, "category": "insurance", "amount_cents": 45000,
		"day_of_month": 15, "every_months": 3, "start_date": "2026-08-15",
	}).decode(t, &re)
	if re.GeneratedThrough != nil || re.NextOn == nil || re.NextOn.String() != "2026-08-15" {
		t.Errorf("future recurring %+v", re)
	}

	var all []recurringView
	c.mustStatus(http.StatusOK, "GET", "/api/recurring-expenses?property_id="+prop.ID, nil).decode(t, &all)
	if len(all) != 2 {
		t.Fatalf("list %+v", all)
	}

	// Stopping it keeps what was already spent.
	c.mustStatus(http.StatusNoContent, "DELETE", "/api/recurring-expenses/"+all[0].ID, nil)
	c.mustStatus(http.StatusOK, "GET", "/api/expenses?property_id="+prop.ID, nil).decode(t, &list)
	if len(list.Expenses) != 3 || list.Expenses[0].RecurringExpenseID != nil {
		t.Errorf("after delete %+v", list.Expenses)
	}
}
