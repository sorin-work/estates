package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type depositStore interface {
	ListDeposits(ctx context.Context, ownerID, leaseID string) ([]store.DepositTransaction, error)
	CreateDeposit(ctx context.Context, ownerID, leaseID string, in store.DepositInput) (string, error)
	DeleteDeposit(ctx context.Context, ownerID, id string) error
	DepositsHeld(ctx context.Context, ownerID string) (map[string]int64, error)
}

func (s *Server) depositRoutes(r chi.Router) {
	r.Delete("/{id}", s.handle(s.deleteDeposit))
}

type depositResponse struct {
	LeaseID string `json:"lease_id"`
	// AgreedCents is the deposit written into the lease.
	AgreedCents int64 `json:"agreed_cents"`
	domain.DepositSummary
	Transactions []store.DepositTransaction `json:"transactions"`
}

func (s *Server) depositSummary(ctx context.Context, ownerID string, l store.Lease) (depositResponse, error) {
	txs, err := s.store.ListDeposits(ctx, ownerID, l.ID)
	if err != nil {
		return depositResponse{}, err
	}
	entries := make([]domain.DepositEntry, len(txs))
	for i, t := range txs {
		entries[i] = domain.DepositEntry{Kind: t.Kind, AmountCents: t.AmountCents}
	}
	return depositResponse{
		LeaseID:        l.ID,
		AgreedCents:    l.DepositCents,
		DepositSummary: domain.SummariseDeposit(entries),
		Transactions:   txs,
	}, nil
}

func (s *Server) leaseDeposit(w http.ResponseWriter, r *http.Request) error {
	l, err := s.loadLease(r)
	if err != nil {
		return err
	}
	resp, err := s.depositSummary(r.Context(), userID(r), l)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

func (s *Server) createDeposit(w http.ResponseWriter, r *http.Request) error {
	l, err := s.loadLease(r)
	if err != nil {
		return err
	}
	var in store.DepositInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	trim(&in.Kind, &in.Notes)

	var v validator
	v.check(domain.OneOf(in.Kind, domain.DepositKinds), "kind", "must be one of "+strings.Join(domain.DepositKinds, ", "))
	v.check(in.AmountCents > 0, "amount_cents", "must be more than zero")
	v.check(!in.OccurredOn.IsZero(), "occurred_on", "is required")
	v.check(!in.OccurredOn.After(s.today()), "occurred_on", "cannot be in the future")
	v.maxLen(in.Notes, "notes", 1000)
	if err := v.err(); err != nil {
		return err
	}

	if _, err := s.store.CreateDeposit(r.Context(), userID(r), l.ID, in); err != nil {
		return notFoundAs(err, "lease")
	}
	s.audit(r, "deposit", l.ID, l.ID, in.Kind, "Deposit "+in.Kind+": "+money(in.AmountCents)+" on "+in.OccurredOn.String())
	resp, err := s.depositSummary(r.Context(), userID(r), l)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, resp)
	return nil
}

func (s *Server) deleteDeposit(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "deposit transaction")
	if err != nil {
		return err
	}
	if err := s.store.DeleteDeposit(r.Context(), userID(r), id); err != nil {
		return notFoundAs(err, "deposit transaction")
	}
	s.audit(r, "deposit", id, "", "deleted", "Deleted a deposit entry")
	w.WriteHeader(http.StatusNoContent)
	return nil
}
