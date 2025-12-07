package domain

import "testing"

func TestLeaseStatus(t *testing.T) {
	start := MustParseDate("2026-01-01")
	end := MustParseDate("2026-12-31")
	term := MustParseDate("2026-06-30")

	tests := []struct {
		name string
		asOf string
		term *Date
		want string
	}{
		{"before start", "2025-12-31", nil, LeaseUpcoming},
		{"first day", "2026-01-01", nil, LeaseActive},
		{"last day", "2026-12-31", nil, LeaseActive},
		{"after end", "2027-01-01", nil, LeaseEnded},
		{"on termination day", "2026-06-30", &term, LeaseActive},
		{"after termination", "2026-07-01", &term, LeaseTerminated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LeaseStatus(start, end, tt.term, MustParseDate(tt.asOf)); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestEffectiveEnd(t *testing.T) {
	end := MustParseDate("2026-12-31")
	early := MustParseDate("2026-06-30")
	late := MustParseDate("2027-03-01")
	if got := EffectiveEnd(end, nil); !got.Equal(end) {
		t.Errorf("no termination: %s", got)
	}
	if got := EffectiveEnd(end, &early); !got.Equal(early) {
		t.Errorf("early termination: %s", got)
	}
	if got := EffectiveEnd(end, &late); !got.Equal(end) {
		t.Errorf("termination after end: %s", got)
	}
}

func TestDateRangesOverlap(t *testing.T) {
	d := MustParseDate
	if !DateRangesOverlap(d("2026-01-01"), d("2026-06-30"), d("2026-06-30"), d("2026-12-31")) {
		t.Error("ranges sharing a day overlap")
	}
	if DateRangesOverlap(d("2026-01-01"), d("2026-06-29"), d("2026-06-30"), d("2026-12-31")) {
		t.Error("back to back ranges do not overlap")
	}
}
