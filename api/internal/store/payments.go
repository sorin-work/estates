package store

import (
	"context"
	"time"

	"github.com/sorin-work/estates/api/internal/domain"
)

type Payment struct {
	ID           string      `db:"id" json:"id"`
	LeaseID      string      `db:"lease_id" json:"lease_id"`
	TenantName   string      `db:"tenant_name" json:"tenant_name"`
	PropertyID   string      `db:"property_id" json:"property_id"`
	PropertyName string      `db:"property_name" json:"property_name"`
	UnitLabel    string      `db:"unit_label" json:"unit_label"`
	AmountCents  int64       `db:"amount_cents" json:"amount_cents"`
	PaidOn       domain.Date `db:"paid_on" json:"paid_on"`
	Method       string      `db:"method" json:"method"`
	Reference    string      `db:"reference" json:"reference"`
	Notes        string      `db:"notes" json:"notes"`
	CreatedAt    time.Time   `db:"created_at" json:"created_at"`
}

type PaymentInput struct {
	AmountCents int64       `json:"amount_cents"`
	PaidOn      domain.Date `json:"paid_on"`
	Method      string      `json:"method"`
	Reference   string      `json:"reference"`
	Notes       string      `json:"notes"`
}

// ExportAll as a PaymentFilter limit returns every matching payment.
const ExportAll = -1

type PaymentFilter struct {
	LeaseID    string
	PropertyID string
	From       domain.Date
	To         domain.Date
	Limit      int
	Offset     int
}

const paymentSelect = `
	SELECT pm.id, pm.lease_id, t.first_name || ' ' || t.last_name AS tenant_name,
	       p.id AS property_id, p.name AS property_name, u.label AS unit_label,
	       pm.amount_cents, pm.paid_on, pm.method, pm.reference, pm.notes, pm.created_at
	  FROM payments pm
	  JOIN leases l ON l.id = pm.lease_id
	  JOIN tenants t ON t.id = l.tenant_id` + ownedLease

// ListPayments returns a page of payments, newest first, and the total
// number matching the filter.
func (s *Store) ListPayments(ctx context.Context, ownerID string, f PaymentFilter) ([]Payment, int, error) {
	w := newFilter(ownerID)
	if f.LeaseID != "" {
		w.add("pm.lease_id = ?", f.LeaseID)
	}
	if f.PropertyID != "" {
		w.add("p.id = ?", f.PropertyID)
	}
	if !f.From.IsZero() {
		w.add("pm.paid_on >= ?", f.From)
	}
	if !f.To.IsZero() {
		w.add("pm.paid_on <= ?", f.To)
	}

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM (`+paymentSelect+w.where()+`) x`, w.args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := paymentSelect + w.where() + ` ORDER BY pm.paid_on DESC, pm.created_at DESC`
	switch {
	case f.Limit == ExportAll:
	case f.Limit <= 0 || f.Limit > 500:
		q += ` LIMIT 50 OFFSET ` + w.arg(max(f.Offset, 0))
	default:
		q += ` LIMIT ` + w.arg(f.Limit) + ` OFFSET ` + w.arg(max(f.Offset, 0))
	}
	rows, err := s.pool.Query(ctx, q, w.args...)
	payments, err := collectAll[Payment](rows, err)
	return payments, total, err
}

func (s *Store) GetPayment(ctx context.Context, ownerID, id string) (Payment, error) {
	rows, err := s.pool.Query(ctx, paymentSelect+` WHERE pm.id = $2`, ownerID, id)
	return collectOne[Payment](rows, err)
}

func (s *Store) CreatePayment(ctx context.Context, ownerID, leaseID string, in PaymentInput) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO payments (lease_id, amount_cents, paid_on, method, reference, notes)
		SELECT l.id, $3, $4, $5, $6, $7 FROM leases l`+ownedLease+` WHERE l.id = $2
		RETURNING id`,
		ownerID, leaseID, in.AmountCents, in.PaidOn, in.Method, in.Reference, in.Notes,
	).Scan(&id)
	return id, notFound(err)
}

func (s *Store) DeletePayment(ctx context.Context, ownerID, id string) error {
	return execOne(s.pool.Exec(ctx, `
		DELETE FROM payments pm USING leases l, units u, properties p
		 WHERE l.id = pm.lease_id AND u.id = l.unit_id AND p.id = u.property_id
		   AND p.owner_id = $1 AND pm.id = $2`, ownerID, id))
}

// LedgerEntries holds the raw charges and payments of one lease.
type LedgerEntries struct {
	Charges  []domain.LedgerCharge
	Payments []domain.LedgerPayment
}

// LedgerEntries loads charges and payments for the owner's leases, keyed by
// lease ID. Pass a lease ID to load a single lease.
func (s *Store) LedgerEntries(ctx context.Context, ownerID, leaseID string) (map[string]*LedgerEntries, error) {
	out := map[string]*LedgerEntries{}
	entry := func(id string) *LedgerEntries {
		e, ok := out[id]
		if !ok {
			e = &LedgerEntries{}
			out[id] = e
		}
		return e
	}

	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.lease_id, c.kind, c.due_date, c.amount_cents
		  FROM charges c
		  JOIN leases l ON l.id = c.lease_id`+ownedLease+`
		 WHERE ($2 = '' OR c.lease_id::text = $2)
		 ORDER BY c.due_date, c.created_at`, ownerID, leaseID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c domain.LedgerCharge
		var lid string
		if err := rows.Scan(&c.ID, &lid, &c.Kind, &c.DueDate, &c.AmountCents); err != nil {
			rows.Close()
			return nil, err
		}
		e := entry(lid)
		e.Charges = append(e.Charges, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `
		SELECT pm.id, pm.lease_id, pm.paid_on, pm.amount_cents
		  FROM payments pm
		  JOIN leases l ON l.id = pm.lease_id`+ownedLease+`
		 WHERE ($2 = '' OR pm.lease_id::text = $2)
		 ORDER BY pm.paid_on`, ownerID, leaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p domain.LedgerPayment
		var lid string
		if err := rows.Scan(&p.ID, &lid, &p.PaidOn, &p.AmountCents); err != nil {
			return nil, err
		}
		e := entry(lid)
		e.Payments = append(e.Payments, p)
	}
	return out, rows.Err()
}
