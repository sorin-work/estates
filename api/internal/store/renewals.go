package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/sorin-work/estates/api/internal/domain"
)

type RentChangeInput struct {
	EffectiveDate domain.Date `json:"effective_date"`
	RentCents     int64       `json:"rent_cents"`
	Notes         string      `json:"notes"`
}

// AddRentChange sets a new rent from a date during the lease. Rent already
// charged for periods from that date is updated to the new amount.
func (s *Store) AddRentChange(ctx context.Context, ownerID, leaseID string, in RentChangeInput) (string, error) {
	var id string
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		l, err := s.getLeaseTx(ctx, tx, ownerID, leaseID)
		if err != nil {
			return err
		}
		end := domain.EffectiveEnd(l.EndDate, l.TerminatedOn)
		if !in.EffectiveDate.After(l.StartDate) || in.EffectiveDate.After(end) {
			return conflict("a rent change must fall after the start and before the end of the lease")
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO rent_changes (lease_id, effective_date, rent_cents, notes)
			VALUES ($1, $2, $3, $4)
			RETURNING id`, leaseID, in.EffectiveDate, in.RentCents, in.Notes).Scan(&id)
		if isUniqueViolation(err) {
			return conflict("there is already a rent change on this date")
		}
		if err != nil {
			return err
		}
		return s.resyncRent(ctx, tx, ownerID, leaseID)
	})
	return id, err
}

// DeleteRentChange removes a rent change and puts the charges it affected
// back to the earlier rent.
func (s *Store) DeleteRentChange(ctx context.Context, ownerID, id string) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		var leaseID string
		err := tx.QueryRow(ctx, `
			DELETE FROM rent_changes rc USING leases l, units u, properties p
			 WHERE l.id = rc.lease_id AND u.id = l.unit_id AND p.id = u.property_id
			   AND p.owner_id = $1 AND rc.id = $2
			RETURNING rc.lease_id`, ownerID, id).Scan(&leaseID)
		if err != nil {
			return notFound(err)
		}
		return s.resyncRent(ctx, tx, ownerID, leaseID)
	})
}

func (s *Store) resyncRent(ctx context.Context, tx pgx.Tx, ownerID, leaseID string) error {
	l, err := s.getLeaseTx(ctx, tx, ownerID, leaseID)
	if err != nil {
		return err
	}
	return syncRent(ctx, tx, leaseID, l.Terms())
}

type RenewalInput struct {
	EndDate   domain.Date `json:"end_date"`
	RentCents int64       `json:"rent_cents"`
	Notes     string      `json:"notes"`
}

// RenewLease creates the lease that follows on from an existing one: same
// unit, tenant and payment terms, starting the day after it ends. The
// deposit moves to the new lease.
func (s *Store) RenewLease(ctx context.Context, ownerID, leaseID string, in RenewalInput) (string, error) {
	var id string
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		l, err := s.getLeaseTx(ctx, tx, ownerID, leaseID)
		if err != nil {
			return err
		}
		if l.TerminatedOn != nil {
			return conflict("a lease that was ended early cannot be renewed")
		}
		if l.RenewedByID != nil {
			return conflict("this lease has already been renewed")
		}
		next := LeaseInput{
			UnitID: l.UnitID, TenantID: l.TenantID,
			StartDate: l.EndDate.AddDays(1), EndDate: in.EndDate,
			RentCents: in.RentCents, DepositCents: l.DepositCents,
			DueDay: l.DueDay, GraceDays: l.GraceDays, LateFeeCents: l.LateFeeCents,
			Notes: in.Notes,
		}
		if next.EndDate.Before(next.StartDate) {
			return conflict("the renewal must end after " + next.StartDate.String())
		}
		if err := checkLeaseRefs(ctx, tx, ownerID, "", next); err != nil {
			return err
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO leases (unit_id, tenant_id, start_date, end_date, rent_cents, deposit_cents,
			                    due_day, grace_days, late_fee_cents, notes, renewed_from_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id`,
			next.UnitID, next.TenantID, next.StartDate, next.EndDate, next.RentCents, next.DepositCents,
			next.DueDay, next.GraceDays, next.LateFeeCents, next.Notes, leaseID,
		).Scan(&id)
		if isUniqueViolation(err) {
			return conflict("this lease has already been renewed")
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE deposit_transactions SET lease_id = $2 WHERE lease_id = $1`, leaseID, id)
		return err
	})
	return id, err
}
