package domain

// RentTerms are the parts of a lease that decide when rent is due and how
// much.
type RentTerms struct {
	Start     Date
	End       Date
	RentCents int64
	// DueDay is the day of the month rent is due, 1-31. Months shorter than
	// DueDay use their last day instead, so 31 means "end of the month".
	DueDay int
	// Changes are rent increases (or decreases) during the lease, in date
	// order. See RentOn.
	Changes []RentChange
}

// RentChange sets a new rent from a date.
type RentChange struct {
	ID            string `json:"id"`
	EffectiveDate Date   `json:"effective_date"`
	RentCents     int64  `json:"rent_cents"`
	Notes         string `json:"notes"`
}

// RentOn returns the monthly rent in force on d: the latest change on or
// before d, or the lease's starting rent.
func (t RentTerms) RentOn(d Date) int64 {
	rent := t.RentCents
	for _, c := range t.Changes {
		if c.EffectiveDate.After(d) {
			break
		}
		rent = c.RentCents
	}
	return rent
}

// ScheduledRent is one rent charge a lease should produce.
type ScheduledRent struct {
	DueDate     Date
	PeriodStart Date
	PeriodEnd   Date
	AmountCents int64
	Prorated    bool
}

func dueDateFor(anyDayOfMonth Date, dueDay int) Date {
	y, m := anyDayOfMonth.Year(), anyDayOfMonth.Month()
	return NewDate(y, m, min(dueDay, DaysIn(y, m)))
}

// nextDueDate returns the first due date strictly after d.
func nextDueDate(d Date, dueDay int) Date {
	due := dueDateFor(d, dueDay)
	if due.After(d) {
		return due
	}
	return dueDateFor(d.MonthStart().AddMonths(1), dueDay)
}

// RentSchedule lists the rent charges for a lease that fall due on or before
// through. Periods run from one due date to the day before the next. A first
// period that starts between due dates, and a last period cut short by the
// lease end, are prorated by the number of days covered. The first charge is
// due on the lease start date. Each period is charged at the rent in force
// on its first day, so a change that takes effect mid-period applies from
// the next one.
func RentSchedule(t RentTerms, through Date) []ScheduledRent {
	if t.Start.IsZero() || t.End.IsZero() || t.End.Before(t.Start) || t.RentCents <= 0 || t.DueDay < 1 {
		return nil
	}

	var out []ScheduledRent
	periodStart := t.Start
	for !periodStart.After(t.End) && !periodStart.After(through) {
		next := nextDueDate(periodStart, t.DueDay)
		fullStart := previousDueDate(next, t.DueDay)
		fullDays := fullStart.DaysUntil(next)

		periodEnd := MinDate(next.AddDays(-1), t.End)
		covered := periodStart.DaysUntil(periodEnd) + 1

		rent := t.RentOn(periodStart)
		charge := ScheduledRent{
			DueDate:     periodStart,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			AmountCents: rent,
		}
		if covered < fullDays {
			charge.AmountCents = prorate(rent, covered, fullDays)
			charge.Prorated = true
		}
		out = append(out, charge)
		periodStart = next
	}
	return out
}

// previousDueDate returns the due date one month before the due date d.
func previousDueDate(d Date, dueDay int) Date {
	return dueDateFor(d.MonthStart().AddMonths(-1), dueDay)
}

// prorate returns amount * part / whole, rounded half up.
func prorate(amount int64, part, whole int) int64 {
	if whole <= 0 {
		return amount
	}
	return (amount*int64(part)*2 + int64(whole)) / (int64(whole) * 2)
}
