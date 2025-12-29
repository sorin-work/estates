package domain

import "testing"

func terms(start, end string, rent int64, dueDay int) RentTerms {
	return RentTerms{Start: MustParseDate(start), End: MustParseDate(end), RentCents: rent, DueDay: dueDay}
}

func dueDates(rs []ScheduledRent) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.DueDate.String()
	}
	return out
}

func TestRentScheduleFullMonths(t *testing.T) {
	rs := RentSchedule(terms("2026-01-01", "2026-12-31", 150000, 1), MustParseDate("2026-04-15"))
	got := dueDates(rs)
	want := []string{"2026-01-01", "2026-02-01", "2026-03-01", "2026-04-01"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("charge %d due %s, want %s", i, got[i], want[i])
		}
		if rs[i].AmountCents != 150000 || rs[i].Prorated {
			t.Errorf("charge %d: %+v", i, rs[i])
		}
	}
	if rs[1].PeriodStart.String() != "2026-02-01" || rs[1].PeriodEnd.String() != "2026-02-28" {
		t.Errorf("february period %s..%s", rs[1].PeriodStart, rs[1].PeriodEnd)
	}
}

func TestRentScheduleProratesFirstPeriod(t *testing.T) {
	// Moving in on the 15th of a 31 day month with rent due on the 1st:
	// 17 of 31 days.
	rs := RentSchedule(terms("2026-03-15", "2027-03-14", 155000, 1), MustParseDate("2026-04-30"))
	if len(rs) != 2 {
		t.Fatalf("got %d charges", len(rs))
	}
	first := rs[0]
	if first.DueDate.String() != "2026-03-15" || first.PeriodEnd.String() != "2026-03-31" {
		t.Errorf("first period %s..%s due %s", first.PeriodStart, first.PeriodEnd, first.DueDate)
	}
	if !first.Prorated || first.AmountCents != 85000 {
		t.Errorf("first charge %d prorated=%v, want 85000", first.AmountCents, first.Prorated)
	}
	if rs[1].AmountCents != 155000 || rs[1].DueDate.String() != "2026-04-01" {
		t.Errorf("second charge %+v", rs[1])
	}
}

func TestRentScheduleProratesLastPeriod(t *testing.T) {
	rs := RentSchedule(terms("2026-01-01", "2026-03-10", 300000, 1), MustParseDate("2026-12-31"))
	if len(rs) != 3 {
		t.Fatalf("got %v", dueDates(rs))
	}
	last := rs[2]
	// 10 of 31 days in March.
	if !last.Prorated || last.AmountCents != 96774 || last.PeriodEnd.String() != "2026-03-10" {
		t.Errorf("last charge %+v", last)
	}
}

func TestRentScheduleDueDayClampsToShortMonths(t *testing.T) {
	rs := RentSchedule(terms("2026-01-31", "2026-06-30", 100000, 31), MustParseDate("2026-05-31"))
	want := []string{"2026-01-31", "2026-02-28", "2026-03-31", "2026-04-30", "2026-05-31"}
	got := dueDates(rs)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("charge %d due %s, want %s", i, got[i], want[i])
		}
		if rs[i].Prorated {
			t.Errorf("charge %d should not be prorated: %+v", i, rs[i])
		}
	}
}

func TestRentScheduleStopsAtThrough(t *testing.T) {
	rs := RentSchedule(terms("2026-01-01", "2026-12-31", 100000, 1), MustParseDate("2025-12-31"))
	if len(rs) != 0 {
		t.Errorf("lease has not started, got %v", dueDates(rs))
	}
	rs = RentSchedule(terms("2026-01-01", "2026-12-31", 100000, 1), MustParseDate("2026-01-01"))
	if len(rs) != 1 {
		t.Errorf("rent is due on the start date, got %v", dueDates(rs))
	}
}

func TestRentScheduleInvalidTerms(t *testing.T) {
	for name, tt := range map[string]RentTerms{
		"end before start": terms("2026-05-01", "2026-04-01", 1000, 1),
		"no rent":          terms("2026-01-01", "2026-12-31", 0, 1),
		"no due day":       terms("2026-01-01", "2026-12-31", 1000, 0),
	} {
		if rs := RentSchedule(tt, MustParseDate("2027-01-01")); len(rs) != 0 {
			t.Errorf("%s: expected no charges, got %v", name, dueDates(rs))
		}
	}
}

func TestProrateRoundsHalfUp(t *testing.T) {
	if got := prorate(100, 1, 3); got != 33 {
		t.Errorf("got %d", got)
	}
	if got := prorate(100, 2, 3); got != 67 {
		t.Errorf("got %d", got)
	}
	if got := prorate(5, 1, 2); got != 3 {
		t.Errorf("got %d", got)
	}
}
