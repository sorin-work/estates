package domain

import "testing"

func stCharge(id, date string, cents int64) StatementEntry {
	return StatementEntry{ID: id, Date: MustParseDate(date), Kind: ChargeRent, AmountCents: cents}
}

func stPayment(id, date string, cents int64) StatementEntry {
	return StatementEntry{ID: id, Date: MustParseDate(date), Payment: true, Kind: "bank_transfer", AmountCents: cents}
}

func lineIDs(st Statement) []string {
	var out []string
	for _, l := range st.Lines {
		out = append(out, l.ID)
	}
	return out
}

func TestStatementWholeLease(t *testing.T) {
	entries := []StatementEntry{
		stPayment("p2", "2026-02-03", 100000),
		stCharge("jan", "2026-01-01", 100000),
		stCharge("feb", "2026-02-01", 100000),
		stPayment("p1", "2026-01-01", 100000),
		stCharge("mar", "2026-03-01", 100000),
	}
	st := BuildStatement(entries, MustParseDate("2026-01-01"), MustParseDate("2026-02-28"))
	if st.OpeningCents != 0 || st.ChargedCents != 200000 || st.PaidCents != 200000 || st.ClosingCents != 0 {
		t.Errorf("totals %+v", st)
	}
	// Same-day rent and payment: the charge comes first. March is due after
	// the period and is left out.
	want := []string{"jan", "p1", "feb", "p2"}
	if got := lineIDs(st); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
		t.Fatalf("lines %v, want %v", got, want)
	}
	balances := []int64{100000, 0, 100000, 0}
	for i, l := range st.Lines {
		if l.BalanceCents != balances[i] {
			t.Errorf("line %d balance %d, want %d", i, l.BalanceCents, balances[i])
		}
	}
}

func TestStatementOpeningBalance(t *testing.T) {
	entries := []StatementEntry{
		stCharge("jan", "2026-01-01", 100000),
		stPayment("p1", "2026-01-05", 60000),
		stCharge("feb", "2026-02-01", 100000),
		stPayment("p2", "2026-02-10", 100000),
	}
	st := BuildStatement(entries, MustParseDate("2026-02-01"), MustParseDate("2026-02-28"))
	if st.OpeningCents != 40000 {
		t.Errorf("opening %d, want 40000", st.OpeningCents)
	}
	if st.ClosingCents != 40000 || len(st.Lines) != 2 || st.Lines[0].BalanceCents != 140000 {
		t.Errorf("statement %+v", st)
	}
}

func TestStatementPrepaymentAndCredit(t *testing.T) {
	entries := []StatementEntry{
		stCharge("jan", "2026-01-01", 100000),
		stPayment("p1", "2026-01-01", 300000), // three months up front
		stCharge("feb", "2026-02-01", 100000),
		stCharge("mar", "2026-03-01", 100000),
	}
	st := BuildStatement(entries, MustParseDate("2026-01-01"), MustParseDate("2026-01-31"))
	if st.ClosingCents != -200000 || st.PaidCents != 300000 || st.ChargedCents != 100000 {
		t.Errorf("statement %+v", st)
	}
}

func TestStatementEmptyPeriod(t *testing.T) {
	entries := []StatementEntry{stCharge("jan", "2026-01-01", 100000)}
	st := BuildStatement(entries, MustParseDate("2026-06-01"), MustParseDate("2026-06-30"))
	if st.OpeningCents != 100000 || st.ClosingCents != 100000 || len(st.Lines) != 0 || st.Lines == nil {
		t.Errorf("statement %+v", st)
	}
}
