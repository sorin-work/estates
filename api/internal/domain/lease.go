package domain

// Lease statuses, derived from dates rather than stored.
const (
	LeaseUpcoming   = "upcoming"
	LeaseActive     = "active"
	LeaseEnded      = "ended"
	LeaseTerminated = "terminated"
)

// ExpiringWindowDays is how far ahead the dashboard looks for leases ending.
const ExpiringWindowDays = 60

// LeaseStatus works out where a lease is on asOf. A lease terminated early
// counts as active until its termination date.
func LeaseStatus(start, end Date, terminatedOn *Date, asOf Date) string {
	if terminatedOn != nil && !terminatedOn.IsZero() && asOf.After(*terminatedOn) {
		return LeaseTerminated
	}
	switch {
	case asOf.Before(start):
		return LeaseUpcoming
	case asOf.After(end):
		return LeaseEnded
	default:
		return LeaseActive
	}
}

// EffectiveEnd is the last day the lease is in force.
func EffectiveEnd(end Date, terminatedOn *Date) Date {
	if terminatedOn != nil && !terminatedOn.IsZero() && terminatedOn.Before(end) {
		return *terminatedOn
	}
	return end
}

// DateRangesOverlap reports whether [aStart, aEnd] and [bStart, bEnd] share at
// least one day.
func DateRangesOverlap(aStart, aEnd, bStart, bEnd Date) bool {
	return !aStart.After(bEnd) && !bStart.After(aEnd)
}
