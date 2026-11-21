package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	d, err := ParseDate("2026-02-28")
	if err != nil {
		t.Fatal(err)
	}
	if d.Year() != 2026 || d.Month() != time.February || d.Day() != 28 {
		t.Errorf("got %v", d)
	}
	for _, bad := range []string{"", "2026-2-28", "28/02/2026", "2026-02-30"} {
		if _, err := ParseDate(bad); err == nil {
			t.Errorf("ParseDate(%q) should fail", bad)
		}
	}
}

func TestAddMonthsClampsDay(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"2026-01-31", 1, "2026-02-28"},
		{"2028-01-31", 1, "2028-02-29"},
		{"2026-03-31", -1, "2026-02-28"},
		{"2026-12-15", 1, "2027-01-15"},
		{"2026-05-31", 12, "2027-05-31"},
	}
	for _, tt := range tests {
		got := MustParseDate(tt.in).AddMonths(tt.n).String()
		if got != tt.want {
			t.Errorf("%s + %d months = %s, want %s", tt.in, tt.n, got, tt.want)
		}
	}
}

func TestDaysUntil(t *testing.T) {
	a := MustParseDate("2026-03-01")
	b := MustParseDate("2026-03-31")
	if got := a.DaysUntil(b); got != 30 {
		t.Errorf("got %d, want 30", got)
	}
	if got := b.DaysUntil(a); got != -30 {
		t.Errorf("got %d, want -30", got)
	}
}

func TestMonthBounds(t *testing.T) {
	d := MustParseDate("2028-02-10")
	if d.MonthStart().String() != "2028-02-01" || d.MonthEnd().String() != "2028-02-29" {
		t.Errorf("got %s..%s", d.MonthStart(), d.MonthEnd())
	}
}

func TestDateJSON(t *testing.T) {
	type wrapper struct {
		D  Date  `json:"d"`
		P  *Date `json:"p"`
		Z  Date  `json:"z"`
		NP *Date `json:"np"`
	}
	d := MustParseDate("2026-07-04")
	b, err := json.Marshal(wrapper{D: d, P: &d})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"d":"2026-07-04","p":"2026-07-04","z":null,"np":null}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}

	var w wrapper
	if err := json.Unmarshal([]byte(`{"d":"2026-01-02","p":"","z":null}`), &w); err != nil {
		t.Fatal(err)
	}
	if w.D.String() != "2026-01-02" || !w.P.IsZero() || !w.Z.IsZero() {
		t.Errorf("unexpected %+v", w)
	}
	if err := json.Unmarshal([]byte(`{"d":"01/02/2026"}`), &w); err == nil {
		t.Error("expected an error for a badly formatted date")
	}
}

func TestTodayIn(t *testing.T) {
	now := time.Date(2026, 3, 31, 23, 30, 0, 0, time.UTC)
	for zone, want := range map[string]string{
		"UTC":                 "2026-03-31",
		"Pacific/Auckland":    "2026-04-01",
		"America/Los_Angeles": "2026-03-31",
		"Asia/Tokyo":          "2026-04-01",
	} {
		loc, err := LoadZone(zone)
		if err != nil {
			t.Fatal(err)
		}
		if got := TodayIn(now, loc).String(); got != want {
			t.Errorf("%s: %s, want %s", zone, got, want)
		}
	}
	// Just after midnight in Los Angeles it is morning in UTC.
	la, _ := LoadZone("America/Los_Angeles")
	if got := TodayIn(time.Date(2026, 4, 1, 7, 30, 0, 0, time.UTC), la).String(); got != "2026-04-01" {
		t.Errorf("Los Angeles after midnight: %s", got)
	}
}

func TestLoadZoneRefusesServerZone(t *testing.T) {
	for _, name := range []string{"", "Local", "Mars/Olympus", "../etc/passwd"} {
		if _, err := LoadZone(name); err == nil {
			t.Errorf("%q accepted", name)
		}
	}
}
