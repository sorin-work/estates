package domain

import (
	"sort"
)

// Charge kinds.
const (
	ChargeRent    = "rent"
	ChargeLateFee = "late_fee"
	ChargeDeposit = "deposit"
	ChargeUtility = "utility"
	ChargeOther   = "other"
)

var ChargeKinds = []string{ChargeRent, ChargeLateFee, ChargeDeposit, ChargeUtility, ChargeOther}

// Charge statuses as seen on a given day.
const (
	StatusPaid     = "paid"
	StatusPartial  = "partial"
	StatusDue      = "due"
	StatusOverdue  = "overdue"
	StatusUpcoming = "upcoming"
)

// LedgerCharge is the input view of a charge for ledger calculations.
type LedgerCharge struct {
	ID          string
	Kind        string
	DueDate     Date
	AmountCents int64
}

// LedgerPayment is the input view of a payment.
type LedgerPayment struct {
	ID          string
	PaidOn      Date
	AmountCents int64
}

// ChargeBalance is a charge after payments have been applied to it.
type ChargeBalance struct {
	LedgerCharge
	PaidCents      int64  `json:"paid_cents"`
	RemainingCents int64  `json:"remaining_cents"`
	Status         string `json:"status"`
}

// Ledger summarises what a tenant owes on a lease as of a date.
type Ledger struct {
	Charges []ChargeBalance
	// ChargedCents is the total of charges due on or before AsOf.
	ChargedCents int64
	// PaidCents is the total of all payments recorded, including prepayments.
	PaidCents int64
	// BalanceCents is ChargedCents - PaidCents. Negative means the tenant is
	// in credit.
	BalanceCents int64
	// OverdueCents is the unpaid amount of charges past their grace period.
	OverdueCents int64
	// OldestOverdue is the due date of the oldest overdue charge, if any.
	OldestOverdue Date
	// DaysOverdue counts days since OldestOverdue.
	DaysOverdue int
}

// IsOverdue reports whether a charge due on due with the given remaining
// balance is overdue on asOf. Rent is overdue once the grace period has
// passed: due on the 1st with 5 grace days means overdue from the 7th.
func IsOverdue(due Date, remaining int64, graceDays int, asOf Date) bool {
	return remaining > 0 && asOf.After(due.AddDays(graceDays))
}

// BuildLedger applies payments to charges oldest-first and works out the
// status of every charge on asOf. Payments are pooled rather than matched to
// specific charges, which is how most small landlords think about it: money
// in pays off the oldest debt first.
func BuildLedger(charges []LedgerCharge, payments []LedgerPayment, graceDays int, asOf Date) Ledger {
	sorted := make([]LedgerCharge, len(charges))
	copy(sorted, charges)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].DueDate.Before(sorted[j].DueDate)
	})

	var l Ledger
	var pool int64
	for _, p := range payments {
		pool += p.AmountCents
	}
	l.PaidCents = pool

	for _, c := range sorted {
		applied := min(pool, c.AmountCents)
		pool -= applied
		cb := ChargeBalance{
			LedgerCharge:   c,
			PaidCents:      applied,
			RemainingCents: c.AmountCents - applied,
		}

		switch {
		case cb.RemainingCents == 0:
			cb.Status = StatusPaid
		case c.DueDate.After(asOf):
			cb.Status = StatusUpcoming
		case IsOverdue(c.DueDate, cb.RemainingCents, graceDays, asOf):
			cb.Status = StatusOverdue
		case applied > 0:
			cb.Status = StatusPartial
		default:
			cb.Status = StatusDue
		}

		if !c.DueDate.After(asOf) {
			l.ChargedCents += c.AmountCents
		}
		if cb.Status == StatusOverdue {
			l.OverdueCents += cb.RemainingCents
			if l.OldestOverdue.IsZero() {
				l.OldestOverdue = c.DueDate
			}
		}
		l.Charges = append(l.Charges, cb)
	}

	l.BalanceCents = l.ChargedCents - l.PaidCents
	if !l.OldestOverdue.IsZero() {
		l.DaysOverdue = l.OldestOverdue.DaysUntil(asOf)
	}
	return l
}

// LateFeeWindowDays limits late fees to rent that became overdue recently.
// Without it, entering an existing lease with a few months of history would
// add a late fee for every month before its payments have been recorded.
const LateFeeWindowDays = 30

// LateFeeTargets returns the rent charges that should get a late fee on asOf:
// rent that is overdue, became overdue within the last LateFeeWindowDays,
// and is not already in alreadyCharged (rent charge IDs that have a fee).
func LateFeeTargets(l Ledger, alreadyCharged map[string]bool, graceDays int, asOf Date) []ChargeBalance {
	cutoff := asOf.AddDays(-LateFeeWindowDays)
	var out []ChargeBalance
	for _, c := range l.Charges {
		if c.Kind != ChargeRent || c.Status != StatusOverdue || alreadyCharged[c.ID] {
			continue
		}
		overdueFrom := c.DueDate.AddDays(graceDays + 1)
		if overdueFrom.Before(cutoff) {
			continue
		}
		out = append(out, c)
	}
	return out
}
