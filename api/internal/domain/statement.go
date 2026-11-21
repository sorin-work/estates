package domain

import "sort"

// StatementEntry is a charge or a payment on a lease, as a statement lists
// it. Amounts are positive for both; Payment says which it is.
type StatementEntry struct {
	ID          string `json:"id"`
	Date        Date   `json:"date"`
	Payment     bool   `json:"payment"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	AmountCents int64  `json:"amount_cents"`
}

// StatementLine is an entry with what the tenant owed after it.
type StatementLine struct {
	StatementEntry
	BalanceCents int64 `json:"balance_cents"`
}

// Statement is a lease's account over a period: what was owed at the
// start, every charge and payment in the period, and what was owed at the
// end. A negative balance means the tenant is in credit.
type Statement struct {
	From         Date            `json:"from"`
	To           Date            `json:"to"`
	OpeningCents int64           `json:"opening_cents"`
	ChargedCents int64           `json:"charged_cents"`
	PaidCents    int64           `json:"paid_cents"`
	ClosingCents int64           `json:"closing_cents"`
	Lines        []StatementLine `json:"lines"`
}

// BuildStatement works out the statement for from to to, inclusive. Charges
// count from their due date and payments from the day they were paid, so a
// charge falling due after to is left out while a payment made in advance
// is not. On the same day charges come before payments, so the running
// balance does not dip for a moment when rent is paid on its due date.
func BuildStatement(entries []StatementEntry, from, to Date) Statement {
	st := Statement{From: from, To: to, Lines: []StatementLine{}}
	var in []StatementEntry
	for _, e := range entries {
		switch {
		case e.Date.Before(from):
			st.OpeningCents += signed(e)
		case !e.Date.After(to):
			in = append(in, e)
		}
	}
	sort.SliceStable(in, func(i, j int) bool {
		if !in[i].Date.Equal(in[j].Date) {
			return in[i].Date.Before(in[j].Date)
		}
		return !in[i].Payment && in[j].Payment
	})

	balance := st.OpeningCents
	for _, e := range in {
		balance += signed(e)
		if e.Payment {
			st.PaidCents += e.AmountCents
		} else {
			st.ChargedCents += e.AmountCents
		}
		st.Lines = append(st.Lines, StatementLine{StatementEntry: e, BalanceCents: balance})
	}
	st.ClosingCents = balance
	return st
}

func signed(e StatementEntry) int64 {
	if e.Payment {
		return -e.AmountCents
	}
	return e.AmountCents
}
