package store

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/sorin-work/estates/api/internal/domain"
)

type Lease struct {
	ID           string       `db:"id" json:"id"`
	UnitID       string       `db:"unit_id" json:"unit_id"`
	UnitLabel    string       `db:"unit_label" json:"unit_label"`
	PropertyID   string       `db:"property_id" json:"property_id"`
	PropertyName string       `db:"property_name" json:"property_name"`
	TenantID     string       `db:"tenant_id" json:"tenant_id"`
	TenantName   string       `db:"tenant_name" json:"tenant_name"`
	StartDate    domain.Date  `db:"start_date" json:"start_date"`
	EndDate      domain.Date  `db:"end_date" json:"end_date"`
	RentCents    int64        `db:"rent_cents" json:"rent_cents"`
	DepositCents int64        `db:"deposit_cents" json:"deposit_cents"`
	DueDay       int          `db:"due_day" json:"due_day"`
	GraceDays    int          `db:"grace_days" json:"grace_days"`
	LateFeeCents int64        `db:"late_fee_cents" json:"late_fee_cents"`
	TerminatedOn *domain.Date `db:"terminated_on" json:"terminated_on"`
	Notes        string       `db:"notes" json:"notes"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
	// Status is derived from the dates; see domain.LeaseStatus.
	Status string `db:"-" json:"status"`
}

// Terms returns the rent terms used to build the rent schedule.
func (l Lease) Terms() domain.RentTerms {
	return domain.RentTerms{
		Start:     l.StartDate,
		End:       domain.EffectiveEnd(l.EndDate, l.TerminatedOn),
		RentCents: l.RentCents,
		DueDay:    l.DueDay,
	}
}

type LeaseInput struct {
	UnitID       string      `json:"unit_id"`
	TenantID     string      `json:"tenant_id"`
	StartDate    domain.Date `json:"start_date"`
	EndDate      domain.Date `json:"end_date"`
	RentCents    int64       `json:"rent_cents"`
	DepositCents int64       `json:"deposit_cents"`
	DueDay       int         `json:"due_day"`
	GraceDays    int         `json:"grace_days"`
	LateFeeCents int64       `json:"late_fee_cents"`
	Notes        string      `json:"notes"`
}

type LeaseFilter struct {
	PropertyID string
	UnitID     string
	TenantID   string
}

const leaseSelect = `
	SELECT l.id, l.unit_id, u.label AS unit_label, p.id AS property_id, p.name AS property_name,
	       l.tenant_id, t.first_name || ' ' || t.last_name AS tenant_name,
	       l.start_date, l.end_date, l.rent_cents, l.deposit_cents, l.due_day, l.grace_days,
	       l.late_fee_cents, l.terminated_on, l.notes, l.created_at
	  FROM leases l
	  JOIN units u ON u.id = l.unit_id
	  JOIN properties p ON p.id = u.property_id
	  JOIN tenants t ON t.id = l.tenant_id`

func (s *Store) ListLeases(ctx context.Context, ownerID string, f LeaseFilter) ([]Lease, error) {
	where := []string{"p.owner_id = $1"}
	args := []any{ownerID}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", "$"+itoa(len(args))))
	}
	if f.PropertyID != "" {
		add("p.id = ?", f.PropertyID)
	}
	if f.UnitID != "" {
		add("l.unit_id = ?", f.UnitID)
	}
	if f.TenantID != "" {
		add("l.tenant_id = ?", f.TenantID)
	}
	q := leaseSelect + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY l.start_date DESC, lower(p.name), u.label`
	rows, err := s.pool.Query(ctx, q, args...)
	return collectAll[Lease](rows, err)
}

func (s *Store) GetLease(ctx context.Context, ownerID, id string) (Lease, error) {
	rows, err := s.pool.Query(ctx, leaseSelect+` WHERE p.owner_id = $1 AND l.id = $2`, ownerID, id)
	return collectOne[Lease](rows, err)
}

// checkLeaseRefs locks the unit and makes sure the unit and tenant belong to
// the owner and that no other lease on the unit overlaps the new dates.
func checkLeaseRefs(ctx context.Context, tx pgx.Tx, ownerID, leaseID string, in LeaseInput) error {
	var unitOK, tenantOK bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		         SELECT 1 FROM units u JOIN properties p ON p.id = u.property_id
		          WHERE u.id = $2 AND p.owner_id = $1),
		       EXISTS (SELECT 1 FROM tenants WHERE id = $3 AND owner_id = $1)`,
		ownerID, in.UnitID, in.TenantID,
	).Scan(&unitOK, &tenantOK)
	if err != nil {
		return err
	}
	if !unitOK {
		return conflict("unit not found")
	}
	if !tenantOK {
		return conflict("tenant not found")
	}

	if _, err := tx.Exec(ctx, `SELECT 1 FROM units WHERE id = $1 FOR UPDATE`, in.UnitID); err != nil {
		return err
	}
	var overlapping bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM leases
		     WHERE unit_id = $1
		       AND id IS DISTINCT FROM $2::uuid
		       AND start_date <= $4
		       AND LEAST(end_date, COALESCE(terminated_on, end_date)) >= $3)`,
		in.UnitID, nullIfEmpty(leaseID), in.StartDate, in.EndDate,
	).Scan(&overlapping)
	if err != nil {
		return err
	}
	if overlapping {
		return conflict("the unit already has a lease for some of these dates")
	}
	return nil
}

func (s *Store) CreateLease(ctx context.Context, ownerID string, in LeaseInput) (string, error) {
	var id string
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := checkLeaseRefs(ctx, tx, ownerID, "", in); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			INSERT INTO leases (unit_id, tenant_id, start_date, end_date, rent_cents, deposit_cents,
			                    due_day, grace_days, late_fee_cents, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id`,
			in.UnitID, in.TenantID, in.StartDate, in.EndDate, in.RentCents, in.DepositCents,
			in.DueDay, in.GraceDays, in.LateFeeCents, in.Notes,
		).Scan(&id)
	})
	return id, err
}

func (s *Store) UpdateLease(ctx context.Context, ownerID, id string, in LeaseInput) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := s.getLeaseTx(ctx, tx, ownerID, id); err != nil {
			return err
		}
		if err := checkLeaseRefs(ctx, tx, ownerID, id, in); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE leases
			   SET unit_id = $2, tenant_id = $3, start_date = $4, end_date = $5, rent_cents = $6,
			       deposit_cents = $7, due_day = $8, grace_days = $9, late_fee_cents = $10,
			       notes = $11, updated_at = now()
			 WHERE id = $1`,
			id, in.UnitID, in.TenantID, in.StartDate, in.EndDate, in.RentCents, in.DepositCents,
			in.DueDay, in.GraceDays, in.LateFeeCents, in.Notes,
		)
		return err
	})
}

// TerminateLease ends a lease early.
func (s *Store) TerminateLease(ctx context.Context, ownerID, id string, on domain.Date) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		l, err := s.getLeaseTx(ctx, tx, ownerID, id)
		if err != nil {
			return err
		}
		if on.Before(l.StartDate) {
			return conflict("termination date is before the lease starts")
		}
		_, err = tx.Exec(ctx, `UPDATE leases SET terminated_on = $2, updated_at = now() WHERE id = $1`, id, on)
		return err
	})
}

// DeleteLease removes a lease.
func (s *Store) DeleteLease(ctx context.Context, ownerID, id string) error {
	return execOne(s.pool.Exec(ctx, `
		DELETE FROM leases l USING units u, properties p
		 WHERE u.id = l.unit_id AND p.id = u.property_id AND p.owner_id = $1 AND l.id = $2`,
		ownerID, id))
}

func (s *Store) getLeaseTx(ctx context.Context, tx pgx.Tx, ownerID, id string) (Lease, error) {
	rows, err := tx.Query(ctx, leaseSelect+` WHERE p.owner_id = $1 AND l.id = $2 FOR UPDATE OF l`, ownerID, id)
	return collectOne[Lease](rows, err)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
