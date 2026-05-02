package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/sorin-work/estates/api/internal/domain"
)

type DepositTransaction struct {
	ID          string      `db:"id" json:"id"`
	LeaseID     string      `db:"lease_id" json:"lease_id"`
	Kind        string      `db:"kind" json:"kind"`
	AmountCents int64       `db:"amount_cents" json:"amount_cents"`
	OccurredOn  domain.Date `db:"occurred_on" json:"occurred_on"`
	Notes       string      `db:"notes" json:"notes"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
}

type DepositInput struct {
	Kind        string      `json:"kind"`
	AmountCents int64       `json:"amount_cents"`
	OccurredOn  domain.Date `json:"occurred_on"`
	Notes       string      `json:"notes"`
}

const depositSelect = `
	SELECT d.id, d.lease_id, d.kind, d.amount_cents, d.occurred_on, d.notes, d.created_at
	  FROM deposit_transactions d
	  JOIN leases l ON l.id = d.lease_id` + ownedLease

// ListDeposits returns a lease's deposit transactions, oldest first.
func (s *Store) ListDeposits(ctx context.Context, ownerID, leaseID string) ([]DepositTransaction, error) {
	rows, err := s.pool.Query(ctx, depositSelect+` WHERE d.lease_id = $2 ORDER BY d.occurred_on, d.created_at`, ownerID, leaseID)
	return collectAll[DepositTransaction](rows, err)
}

func depositSummaryTx(ctx context.Context, tx pgx.Tx, leaseID string) (domain.DepositSummary, error) {
	rows, err := tx.Query(ctx, `SELECT kind, amount_cents FROM deposit_transactions WHERE lease_id = $1`, leaseID)
	if err != nil {
		return domain.DepositSummary{}, err
	}
	entries, err := pgx.CollectRows(rows, pgx.RowToStructByPos[domain.DepositEntry])
	if err != nil {
		return domain.DepositSummary{}, err
	}
	return domain.SummariseDeposit(entries), nil
}

// CreateDeposit records money received for, returned from or withheld from
// a lease's deposit. More than is held cannot be paid out.
func (s *Store) CreateDeposit(ctx context.Context, ownerID, leaseID string, in DepositInput) (string, error) {
	var id string
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := s.getLeaseTx(ctx, tx, ownerID, leaseID); err != nil {
			return err
		}
		if domain.ReducesDeposit(in.Kind) {
			sum, err := depositSummaryTx(ctx, tx, leaseID)
			if err != nil {
				return err
			}
			if in.AmountCents > sum.HeldCents {
				return conflict("that is more than the deposit held for this lease")
			}
		}
		return tx.QueryRow(ctx, `
			INSERT INTO deposit_transactions (lease_id, kind, amount_cents, occurred_on, notes)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			leaseID, in.Kind, in.AmountCents, in.OccurredOn, in.Notes,
		).Scan(&id)
	})
	return id, err
}

// DeleteDeposit removes a deposit transaction. A received deposit that has
// already been paid back out cannot be removed.
func (s *Store) DeleteDeposit(ctx context.Context, ownerID, id string) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		var leaseID, kind string
		var amount int64
		err := tx.QueryRow(ctx, `
			SELECT d.lease_id, d.kind, d.amount_cents
			  FROM deposit_transactions d
			  JOIN leases l ON l.id = d.lease_id`+ownedLease+`
			 WHERE d.id = $2
			   FOR UPDATE OF l`, ownerID, id).Scan(&leaseID, &kind, &amount)
		if err != nil {
			return notFound(err)
		}
		if kind == domain.DepositReceived {
			sum, err := depositSummaryTx(ctx, tx, leaseID)
			if err != nil {
				return err
			}
			if sum.HeldCents-amount < 0 {
				return conflict("part of this deposit has already been returned or withheld")
			}
		}
		_, err = tx.Exec(ctx, `DELETE FROM deposit_transactions WHERE id = $1`, id)
		return err
	})
}

// DepositsHeld returns the deposit held for each of the owner's leases that
// holds one.
func (s *Store) DepositsHeld(ctx context.Context, ownerID string) (map[string]int64, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.lease_id::text,
		       sum(CASE WHEN d.kind = 'received' THEN d.amount_cents ELSE -d.amount_cents END)::bigint
		  FROM deposit_transactions d
		  JOIN leases l ON l.id = d.lease_id`+ownedLease+`
		 GROUP BY d.lease_id
		HAVING sum(CASE WHEN d.kind = 'received' THEN d.amount_cents ELSE -d.amount_cents END) <> 0`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var id string
		var held int64
		if err := rows.Scan(&id, &held); err != nil {
			return nil, err
		}
		out[id] = held
	}
	return out, rows.Err()
}
