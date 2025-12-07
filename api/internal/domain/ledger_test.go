package domain

import "testing"

func rent(id, due string, amount int64) LedgerCharge {
	return LedgerCharge{ID: id, Kind: ChargeRent, DueDate: MustParseDate(due), AmountCents: amount}
}

func payment(on string, amount int64) LedgerPayment {
	return LedgerPayment{ID: "p-" + on, PaidOn: MustParseDate(on), AmountCents: amount}
}

func statuses(l Ledger) []string {
	out := make([]string, len(l.Charges))
	for i, c := range l.Charges {
		out[i] = c.Status
	}
	return out
}

func TestLedgerAllPaid(t *testing.T) {
	l := BuildLedger(
		[]LedgerCharge{rent("jan", "2026-01-01", 1000), rent("feb", "2026-02-01", 1000)},
		[]LedgerPayment{payment("2026-01-02", 1000), payment("2026-02-01", 1000)},
		5, MustParseDate("2026-02-20"),
	)
	if l.BalanceCents != 0 || l.OverdueCents != 0 || l.DaysOverdue != 0 {
		t.Errorf("unexpected %+v", l)
	}
	for _, s := range statuses(l) {
		if s != StatusPaid {
			t.Errorf("status %s", s)
		}
	}
}

func TestLedgerAppliesPaymentsOldestFirst(t *testing.T) {
	// Charges are given out of order on purpose.
	l := BuildLedger(
		[]LedgerCharge{rent("mar", "2026-03-01", 1000), rent("jan", "2026-01-01", 1000), rent("feb", "2026-02-01", 1000)},
		[]LedgerPayment{payment("2026-03-02", 1500)},
		5, MustParseDate("2026-03-03"),
	)
	got := statuses(l)
	want := []string{StatusPaid, StatusOverdue, StatusDue}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("statuses %v, want %v", got, want)
		}
	}
	if l.Charges[1].PaidCents != 500 || l.Charges[1].RemainingCents != 500 {
		t.Errorf("february %+v", l.Charges[1])
	}
	if l.BalanceCents != 1500 || l.OverdueCents != 500 {
		t.Errorf("balance %d overdue %d", l.BalanceCents, l.OverdueCents)
	}
	if l.OldestOverdue.String() != "2026-02-01" || l.DaysOverdue != 30 {
		t.Errorf("oldest overdue %s, %d days", l.OldestOverdue, l.DaysOverdue)
	}
}

func TestLedgerGracePeriod(t *testing.T) {
	charges := []LedgerCharge{rent("jan", "2026-01-01", 1000)}
	// Due on the 1st with 5 days grace: still fine on the 6th, overdue on the 7th.
	if s := statuses(BuildLedger(charges, nil, 5, MustParseDate("2026-01-06")))[0]; s != StatusDue {
		t.Errorf("on the 6th: %s", s)
	}
	if s := statuses(BuildLedger(charges, nil, 5, MustParseDate("2026-01-07")))[0]; s != StatusOverdue {
		t.Errorf("on the 7th: %s", s)
	}
	partial := BuildLedger(charges, []LedgerPayment{payment("2026-01-02", 400)}, 5, MustParseDate("2026-01-03"))
	if s := statuses(partial)[0]; s != StatusPartial {
		t.Errorf("partly paid: %s", s)
	}
}

func TestLedgerFutureChargesAndCredit(t *testing.T) {
	l := BuildLedger(
		[]LedgerCharge{rent("jan", "2026-01-01", 1000), rent("feb", "2026-02-01", 1000)},
		[]LedgerPayment{payment("2026-01-01", 1200)},
		5, MustParseDate("2026-01-15"),
	)
	if got := statuses(l); got[0] != StatusPaid || got[1] != StatusUpcoming {
		t.Errorf("statuses %v", got)
	}
	// Only January is due so far, and 1200 was paid: 200 in credit.
	if l.ChargedCents != 1000 || l.BalanceCents != -200 {
		t.Errorf("charged %d balance %d", l.ChargedCents, l.BalanceCents)
	}
	if l.Charges[1].PaidCents != 200 {
		t.Errorf("prepayment should go to february, got %+v", l.Charges[1])
	}
}

func TestLedgerPaymentBeforeAnyCharge(t *testing.T) {
	// A deposit-style payment recorded before rent starts still counts.
	l := BuildLedger(
		[]LedgerCharge{rent("jan", "2026-01-01", 1000)},
		[]LedgerPayment{payment("2025-12-20", 1000)},
		0, MustParseDate("2026-01-05"),
	)
	if l.BalanceCents != 0 || statuses(l)[0] != StatusPaid {
		t.Errorf("unexpected %+v", l)
	}
}
