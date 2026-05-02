package domain

// Deposit transaction kinds. A deposit is received when the tenant moves in
// and at the end is returned, or partly withheld to cover damage or unpaid
// rent.
const (
	DepositReceived = "received"
	DepositReturned = "returned"
	DepositWithheld = "withheld"
)

var DepositKinds = []string{DepositReceived, DepositReturned, DepositWithheld}

// DepositEntry is the input view of a deposit transaction.
type DepositEntry struct {
	Kind        string
	AmountCents int64
}

// DepositSummary totals a lease's deposit transactions.
type DepositSummary struct {
	ReceivedCents int64 `json:"received_cents"`
	ReturnedCents int64 `json:"returned_cents"`
	WithheldCents int64 `json:"withheld_cents"`
	// HeldCents is what the landlord still holds for the tenant.
	HeldCents int64 `json:"held_cents"`
}

// SummariseDeposit adds up deposit transactions.
func SummariseDeposit(entries []DepositEntry) DepositSummary {
	var s DepositSummary
	for _, e := range entries {
		switch e.Kind {
		case DepositReceived:
			s.ReceivedCents += e.AmountCents
		case DepositReturned:
			s.ReturnedCents += e.AmountCents
		case DepositWithheld:
			s.WithheldCents += e.AmountCents
		}
	}
	s.HeldCents = s.ReceivedCents - s.ReturnedCents - s.WithheldCents
	return s
}

// ReducesDeposit reports whether a transaction kind pays money out of the
// held deposit.
func ReducesDeposit(kind string) bool {
	return kind == DepositReturned || kind == DepositWithheld
}
