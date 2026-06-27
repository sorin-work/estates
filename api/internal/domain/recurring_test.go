package domain

import (
	"strings"
	"testing"
)

func dates(ds []Date) string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.String()
	}
	return strings.Join(out, " ")
}

func TestOccurrences(t *testing.T) {
	d := MustParseDate
	tests := []struct {
		name    string
		r       Recurrence
		after   string
		through string
		want    string
	}{
		{"monthly from the start day", Recurrence{Start: d("2026-01-01"), Day: 1, EveryMonths: 1}, "", "2026-03-15",
			"2026-01-01 2026-02-01 2026-03-01"},
		{"starts after this month's day", Recurrence{Start: d("2026-01-20"), Day: 15, EveryMonths: 1}, "", "2026-03-31",
			"2026-02-15 2026-03-15"},
		{"end of month clamps", Recurrence{Start: d("2026-01-01"), Day: 31, EveryMonths: 1}, "", "2026-04-30",
			"2026-01-31 2026-02-28 2026-03-31 2026-04-30"},
		{"quarterly", Recurrence{Start: d("2026-01-10"), Day: 10, EveryMonths: 3}, "", "2026-12-31",
			"2026-01-10 2026-04-10 2026-07-10 2026-10-10"},
		{"yearly", Recurrence{Start: d("2025-03-01"), Day: 1, EveryMonths: 12}, "", "2027-06-01",
			"2025-03-01 2026-03-01 2027-03-01"},
		{"stops at the end date", Recurrence{Start: d("2026-01-05"), End: d("2026-03-04"), Day: 5, EveryMonths: 1}, "", "2026-12-31",
			"2026-01-05 2026-02-05"},
		{"only after a date", Recurrence{Start: d("2026-01-01"), Day: 1, EveryMonths: 1}, "2026-02-01", "2026-04-01",
			"2026-03-01 2026-04-01"},
		{"nothing yet", Recurrence{Start: d("2026-05-01"), Day: 1, EveryMonths: 1}, "", "2026-04-30", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var after Date
			if tt.after != "" {
				after = d(tt.after)
			}
			if got := dates(tt.r.Occurrences(after, d(tt.through))); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNextOccurrence(t *testing.T) {
	d := MustParseDate
	r := Recurrence{Start: d("2026-01-15"), End: d("2026-06-30"), Day: 15, EveryMonths: 3}
	if got := r.Next(d("2026-01-15")); got.String() != "2026-04-15" {
		t.Errorf("next after the first: %s", got)
	}
	if got := r.Next(d("2025-06-01")); got.String() != "2026-01-15" {
		t.Errorf("next before the start: %s", got)
	}
	if got := r.Next(d("2026-04-15")); !got.IsZero() {
		t.Errorf("next after the end: %s", got)
	}
}
