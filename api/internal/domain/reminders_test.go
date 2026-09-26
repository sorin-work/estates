package domain

import "testing"

func TestReminders(t *testing.T) {
	d := MustParseDate
	today := d("2026-10-10")
	ended := d("2026-09-30")
	leases := []LeaseFacts{
		// Ends in 21 days, not renewed.
		{ID: "ending", TenantName: "Kevin O'Brien", Place: "14 Birch Lane", Start: d("2024-11-01"), End: d("2026-10-31"),
			Terms: RentTerms{Changes: []RentChange{{EffectiveDate: d("2026-10-20"), RentCents: 1}}}},
		// Ends in 45 days but has been renewed.
		{ID: "renewed", TenantName: "Priya Raman", Place: "Harbor 101", Start: d("2025-11-25"), End: d("2026-11-24"), Renewed: true},
		// Ends in 80 days.
		{ID: "later", TenantName: "Maria Gonzalez", Place: "Maple A", Start: d("2025-01-01"), End: d("2026-12-29")},
		// Moved out ten days ago and the deposit is still held.
		{ID: "moved-out", TenantName: "Marcus Reed", Place: "Harbor 203", Start: d("2025-10-01"), End: d("2026-12-31"),
			TerminatedOn: &ended, DepositHeld: 168000},
		// Moved out but the deposit was settled.
		{ID: "settled", TenantName: "A B", Place: "X", Start: d("2025-01-01"), End: d("2026-08-31")},
		// Moving in next week without a deposit recorded.
		{ID: "moving-in", TenantName: "Elena Petrova", Place: "Harbor 203", Start: d("2026-10-15"), End: d("2027-10-14"), DepositCents: 172500},
	}
	requests := []MaintenanceFacts{
		{ID: "leak", Title: "Water leaking", Priority: "urgent", Status: MaintenanceInProgress, ReportedOn: d("2026-10-07")},
		{ID: "fresh", Title: "Hall light", Priority: "urgent", Status: MaintenanceOpen, ReportedOn: d("2026-10-09")},
		{ID: "done", Title: "Boiler", Priority: "high", Status: MaintenanceCompleted, ReportedOn: d("2026-08-01")},
	}

	got := Reminders(today, leases, requests, 30)
	type key struct{ kind, id, sev string }
	want := []key{
		{ReminderLeaseEnding, "ending", SeverityUrgent},
		{ReminderMaintenance, "leak", SeverityWarning},
		{ReminderDepositReturn, "moved-out", SeverityWarning},
		{ReminderMoveIn, "moving-in", SeverityInfo},
		{ReminderRentIncrease, "ending", SeverityInfo},
		{ReminderLeaseEnding, "later", SeverityInfo},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d reminders: %+v", len(got), got)
	}
	for i, w := range want {
		id := got[i].LeaseID + got[i].MaintenanceID
		if got[i].Kind != w.kind || id != w.id || got[i].Severity != w.sev {
			t.Errorf("%d: got %s %s %s, want %v", i, got[i].Kind, id, got[i].Severity, w)
		}
	}
	if got[0].Title != "Lease ends in 21 days" {
		t.Errorf("title %q", got[0].Title)
	}
	if got[2].Title != "Return the deposit in 20 days" || got[2].Date.String() != "2026-10-30" {
		t.Errorf("deposit reminder %+v", got[2])
	}
	if got[1].Detail != "An urgent request still open after 3 days." {
		t.Errorf("maintenance detail %q", got[1].Detail)
	}
	if got[3].Detail != "Elena Petrova at Harbor 203. The deposit has not been recorded yet." {
		t.Errorf("move-in detail %q", got[3].Detail)
	}
}

func TestLateDepositReturnIsUrgent(t *testing.T) {
	d := MustParseDate
	l := LeaseFacts{ID: "x", TenantName: "T", Place: "P", Start: d("2025-01-01"), End: d("2026-06-30"), DepositHeld: 100}
	got := Reminders(d("2026-08-02"), []LeaseFacts{l}, nil, 30)
	if len(got) != 1 || got[0].Severity != SeverityUrgent || got[0].Title != "Deposit return is 3 days late" {
		t.Errorf("got %+v", got)
	}
}
