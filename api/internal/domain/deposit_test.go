package domain

import "testing"

func TestSummariseDeposit(t *testing.T) {
	s := SummariseDeposit([]DepositEntry{
		{Kind: DepositReceived, AmountCents: 150000},
		{Kind: DepositReceived, AmountCents: 25000},
		{Kind: DepositWithheld, AmountCents: 30000},
		{Kind: DepositReturned, AmountCents: 100000},
	})
	want := DepositSummary{ReceivedCents: 175000, ReturnedCents: 100000, WithheldCents: 30000, HeldCents: 45000}
	if s != want {
		t.Errorf("got %+v, want %+v", s, want)
	}
	if empty := SummariseDeposit(nil); empty != (DepositSummary{}) {
		t.Errorf("empty %+v", empty)
	}
}

func TestReducesDeposit(t *testing.T) {
	if ReducesDeposit(DepositReceived) || !ReducesDeposit(DepositReturned) || !ReducesDeposit(DepositWithheld) {
		t.Error("only returns and withholdings reduce the deposit")
	}
}
