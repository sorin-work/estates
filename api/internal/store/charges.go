package store

import (
	"context"
	"time"

	"github.com/sorin-work/estates/api/internal/domain"
)

type Charge struct {
	ID             string       `db:"id" json:"id"`
	LeaseID        string       `db:"lease_id" json:"lease_id"`
	Kind           string       `db:"kind" json:"kind"`
	Description    string       `db:"description" json:"description"`
	AmountCents    int64        `db:"amount_cents" json:"amount_cents"`
	DueDate        domain.Date  `db:"due_date" json:"due_date"`
	PeriodStart    *domain.Date `db:"period_start" json:"period_start"`
	PeriodEnd      *domain.Date `db:"period_end" json:"period_end"`
	ParentChargeID *string      `db:"parent_charge_id" json:"parent_charge_id"`
	CreatedAt      time.Time    `db:"created_at" json:"created_at"`
}

type ChargeInput struct {
	Kind        string      `json:"kind"`
	Description string      `json:"description"`
	AmountCents int64       `json:"amount_cents"`
	DueDate     domain.Date `json:"due_date"`
}

const chargeColumns = `c.id, c.lease_id, c.kind, c.description, c.amount_cents, c.due_date,
	c.period_start, c.period_end, c.parent_charge_id, c.created_at`

// ownedLease is a join fragment restricting a lease alias l to owner $1.
const ownedLease = `
	JOIN units u ON u.id = l.unit_id
	JOIN properties p ON p.id = u.property_id AND p.owner_id = $1`

func (s *Store) ListCharges(ctx context.Context, ownerID, leaseID string) ([]Charge, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+chargeColumns+`
		  FROM charges c
		  JOIN leases l ON l.id = c.lease_id`+ownedLease+`
		 WHERE c.lease_id = $2
		 ORDER BY c.due_date, c.created_at`, ownerID, leaseID)
	return collectAll[Charge](rows, err)
}

// CreateCharge adds a one-off charge such as a deposit or utility bill.
func (s *Store) CreateCharge(ctx context.Context, ownerID, leaseID string, in ChargeInput) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO charges (lease_id, kind, description, amount_cents, due_date)
		SELECT l.id, $3, $4, $5, $6 FROM leases l`+ownedLease+` WHERE l.id = $2
		RETURNING id`,
		ownerID, leaseID, in.Kind, in.Description, in.AmountCents, in.DueDate,
	).Scan(&id)
	if isUniqueViolation(err) {
		return "", conflict("rent is already charged for this date")
	}
	return id, notFound(err)
}

// DeleteCharge removes a one-off charge. Rent comes from the lease terms
// and billing would only add it again, so it cannot be deleted; change the
// lease instead.
func (s *Store) DeleteCharge(ctx context.Context, ownerID, id string) error {
	var kind string
	err := s.pool.QueryRow(ctx, `
		SELECT c.kind FROM charges c
		  JOIN leases l ON l.id = c.lease_id`+ownedLease+`
		 WHERE c.id = $2`, ownerID, id).Scan(&kind)
	if err != nil {
		return notFound(err)
	}
	if kind == domain.ChargeRent {
		return conflict("rent comes from the lease terms; edit or end the lease instead")
	}
	return execOne(s.pool.Exec(ctx, `DELETE FROM charges WHERE id = $1`, id))
}

// InsertRent adds scheduled rent charges, skipping periods that already have
// one. It returns how many were added.
func (s *Store) InsertRent(ctx context.Context, leaseID string, rent []domain.ScheduledRent) (int, error) {
	added := 0
	for _, r := range rent {
		desc := rentDescription(r)
		tag, err := s.pool.Exec(ctx, `
			INSERT INTO charges (lease_id, kind, description, amount_cents, due_date, period_start, period_end)
			VALUES ($1, 'rent', $2, $3, $4, $5, $6)
			ON CONFLICT (lease_id, due_date) WHERE kind = 'rent' DO NOTHING`,
			leaseID, desc, r.AmountCents, r.DueDate, r.PeriodStart, r.PeriodEnd)
		if err != nil {
			return added, err
		}
		added += int(tag.RowsAffected())
	}
	return added, nil
}

func rentDescription(r domain.ScheduledRent) string {
	desc := "Rent " + r.PeriodStart.String() + " to " + r.PeriodEnd.String()
	if r.Prorated {
		desc += " (prorated)"
	}
	return desc
}

// InsertLateFee adds a late fee for an overdue rent charge unless it already
// has one. It reports whether a fee was added.
func (s *Store) InsertLateFee(ctx context.Context, leaseID, rentChargeID string, amount int64, due domain.Date) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO charges (lease_id, kind, description, amount_cents, due_date, parent_charge_id)
		SELECT $1, 'late_fee', 'Late fee for rent due ' || to_char(c.due_date, 'YYYY-MM-DD'), $3, $4, c.id
		  FROM charges c WHERE c.id = $2
		ON CONFLICT (parent_charge_id) WHERE kind = 'late_fee' DO NOTHING`,
		leaseID, rentChargeID, amount, due)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// LateFeeParents returns the IDs of rent charges on a lease that already
// have a late fee.
func (s *Store) LateFeeParents(ctx context.Context, leaseID string) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT parent_charge_id::text FROM charges
		 WHERE lease_id = $1 AND kind = 'late_fee' AND parent_charge_id IS NOT NULL`, leaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
