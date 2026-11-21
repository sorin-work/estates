package httpapi

import (
	"context"
	"net/http"

	"github.com/sorin-work/estates/api/internal/domain"
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

type statementStore interface {
	StatementPayments(ctx context.Context, ownerID, leaseID string) ([]domain.StatementEntry, error)
}

// leaseStatement is a printable account of a lease over a period. By
// default it runs from the lease start, or from the first charge or
// payment if one came earlier (a deposit or rent paid before move-in), to
// today.
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

	ctx, owner := r.Context(), userID(r)
	charges, err := s.store.ListCharges(ctx, owner, l.ID)
	if err != nil {
		return err
	}
	// Every payment, however many: older ones make the opening balance.
	entries, err := s.store.StatementPayments(ctx, owner, l.ID)
	if err != nil {
		return err
	}
	for _, c := range charges {
		entries = append(entries, domain.StatementEntry{
			ID: c.ID, Date: c.DueDate, Kind: c.Kind, Description: c.Description, AmountCents: c.AmountCents,
		})
	}

	if from.IsZero() {
		from = l.StartDate
		for _, e := range entries {
			if e.Date.Before(from) {
				from = e.Date
			}
		}
	}
	if to.IsZero() {
		to = today
	}
	if to.Before(from) {
		return errBadRequest("to must not be before from")
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
