package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Date is a calendar date without a time of day or timezone. It is stored as
// a Postgres DATE and sent over the wire as "YYYY-MM-DD".
type Date struct {
	t time.Time
}

const dateLayout = "2006-01-02"

// NewDate returns the given calendar date. Out-of-range values are normalised
// the same way time.Date does (e.g. February 30 becomes March 2).
func NewDate(year int, month time.Month, day int) Date {
	return Date{t: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// DateOf returns the calendar date of t in t's own location.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return NewDate(y, m, d)
}

// Today returns the current date in UTC.
func Today() Date {
	return DateOf(time.Now().UTC())
}

// ParseDate parses a "YYYY-MM-DD" string.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", s)
	}
	return Date{t: t}, nil
}

// MustParseDate is ParseDate for tests and fixtures; it panics on bad input.
func MustParseDate(s string) Date {
	d, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func (d Date) Year() int          { return d.t.Year() }
func (d Date) Month() time.Month  { return d.t.Month() }
func (d Date) Day() int           { return d.t.Day() }
func (d Date) IsZero() bool       { return d.t.IsZero() }
func (d Date) Time() time.Time    { return d.t }
func (d Date) Before(o Date) bool { return d.t.Before(o.t) }
func (d Date) After(o Date) bool  { return d.t.After(o.t) }
func (d Date) Equal(o Date) bool  { return d.t.Equal(o.t) }

// Compare returns -1, 0 or 1.
func (d Date) Compare(o Date) int { return d.t.Compare(o.t) }

func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.t.Format(dateLayout)
}

// AddDays returns the date n days later (or earlier for negative n).
func (d Date) AddDays(n int) Date {
	return Date{t: d.t.AddDate(0, 0, n)}
}

// AddMonths moves the date by n months, clamping the day to the length of the
// target month: January 31 + 1 month is February 28 (or 29).
func (d Date) AddMonths(n int) Date {
	first := time.Date(d.Year(), d.Month()+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	day := min(d.Day(), DaysIn(first.Year(), first.Month()))
	return NewDate(first.Year(), first.Month(), day)
}

// DaysUntil returns the number of days from d to o (negative if o is earlier).
func (d Date) DaysUntil(o Date) int {
	return int(o.t.Sub(d.t).Hours() / 24)
}

// MonthStart returns the first day of d's month.
func (d Date) MonthStart() Date { return NewDate(d.Year(), d.Month(), 1) }

// MonthEnd returns the last day of d's month.
func (d Date) MonthEnd() Date {
	return NewDate(d.Year(), d.Month(), DaysIn(d.Year(), d.Month()))
}

// DaysIn returns the number of days in the given month.
func DaysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// MinDate returns the earlier of two dates.
func MinDate(a, b Date) Date {
	if a.Before(b) {
		return a
	}
	return b
}

func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("date must be a string in YYYY-MM-DD format")
	}
	if s == "" {
		*d = Date{}
		return nil
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// ScanDate implements pgtype.DateScanner.
func (d *Date) ScanDate(v pgtype.Date) error {
	if !v.Valid {
		*d = Date{}
		return nil
	}
	*d = DateOf(v.Time)
	return nil
}

// DateValue implements pgtype.DateValuer.
func (d Date) DateValue() (pgtype.Date, error) {
	if d.IsZero() {
		return pgtype.Date{}, nil
	}
	return pgtype.Date{Time: d.t, Valid: true}, nil
}
