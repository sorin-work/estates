package domain

// Recurrence intervals, in months.
var RecurrenceMonths = []int{1, 3, 6, 12}

// Recurrence describes a cost that comes back on a fixed day, such as a
// mortgage payment or an insurance premium.
type Recurrence struct {
	Start Date
	// End is the last day an occurrence may fall on; zero means open ended.
	End Date
	// Day is the day of the month, 1-31. Months shorter than Day use their
	// last day.
	Day int
	// EveryMonths is 1 for monthly, 3 for quarterly and so on.
	EveryMonths int
}

// Occurrences lists the dates of a recurrence that fall after `after` (which
// may be zero) and on or before through. The first occurrence is the first
// matching day on or after the start date; later ones follow every
// EveryMonths months from there.
func (r Recurrence) Occurrences(after, through Date) []Date {
	if r.Start.IsZero() || r.Day < 1 || r.EveryMonths < 1 {
		return nil
	}
	first := dueDateFor(r.Start, r.Day)
	if first.Before(r.Start) {
		first = dueDateFor(r.Start.MonthStart().AddMonths(1), r.Day)
	}
	var out []Date
	for i := 0; ; i++ {
		d := dueDateFor(first.MonthStart().AddMonths(i*r.EveryMonths), r.Day)
		if d.After(through) || (!r.End.IsZero() && d.After(r.End)) {
			return out
		}
		if after.IsZero() || d.After(after) {
			out = append(out, d)
		}
	}
}

// Next returns the first occurrence after `after`, or a zero date if the
// recurrence has ended by then.
func (r Recurrence) Next(after Date) Date {
	// Two intervals ahead is always enough to find the next one.
	through := after.AddMonths(2*r.EveryMonths + 1)
	if after.Before(r.Start) {
		through = r.Start.AddMonths(2*r.EveryMonths + 1)
	}
	for _, d := range r.Occurrences(after, through) {
		return d
	}
	return Date{}
}
