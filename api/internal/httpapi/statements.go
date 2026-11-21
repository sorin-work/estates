package httpapi

import (
	"net/http"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type statementResponse struct {
	domain.Statement
	IssuedOn domain.Date `json:"issued_on"`
	Lease    struct {
		ID        string      `json:"id"`
		StartDate domain.Date `json:"start_date"`
		EndDate   domain.Date `json:"end_date"`
	} `json:"lease"`
	letterhead
}

// leaseStatement is a printable account of a lease over a period, by
// default from its start to today.
func (s *Server) leaseStatement(w http.ResponseWriter, r *http.Request) error {
	l, err := s.loadLease(r)
	if err != nil {
		return err
	}
	today := s.today(r)
	from, err := queryDate(r, "from")
	if err != nil {
		return err
	}
	to, err := queryDate(r, "to")
	if err != nil {
		return err
	}
	if from.IsZero() {
		from = l.StartDate
	}
	if to.IsZero() {
		to = today
	}
	if to.Before(from) {
		return errBadRequest("to must not be before from")
	}

	ctx, owner := r.Context(), userID(r)
	charges, err := s.store.ListCharges(ctx, owner, l.ID)
	if err != nil {
		return err
	}
	// Every payment, like the ledger: older ones make the opening balance.
	payments, _, err := s.store.ListPayments(ctx, owner, store.PaymentFilter{LeaseID: l.ID, Limit: 500})
	if err != nil {
		return err
	}
	entries := make([]domain.StatementEntry, 0, len(charges)+len(payments))
	for _, c := range charges {
		entries = append(entries, domain.StatementEntry{
			ID: c.ID, Date: c.DueDate, Kind: c.Kind, Description: c.Description, AmountCents: c.AmountCents,
		})
	}
	for _, p := range payments {
		entries = append(entries, domain.StatementEntry{
			ID: p.ID, Date: p.PaidOn, Payment: true, Kind: p.Method, Description: p.Reference, AmountCents: p.AmountCents,
		})
	}

	head, err := s.loadLetterhead(r, l)
	if err != nil {
		return err
	}
	resp := statementResponse{Statement: domain.BuildStatement(entries, from, to), IssuedOn: today, letterhead: head}
	resp.Lease.ID, resp.Lease.StartDate, resp.Lease.EndDate = l.ID, l.StartDate, l.EndDate
	writeJSON(w, http.StatusOK, resp)
	return nil
}
