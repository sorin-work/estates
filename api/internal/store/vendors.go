package store

import (
	"context"
	"time"

	"github.com/sorin-work/estates/api/internal/domain"
)

type Vendor struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Trade       string    `db:"trade" json:"trade"`
	ContactName string    `db:"contact_name" json:"contact_name"`
	Phone       string    `db:"phone" json:"phone"`
	Email       string    `db:"email" json:"email"`
	Notes       string    `db:"notes" json:"notes"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	// Totals across the vendor's jobs and bills.
	OpenJobs   int          `db:"open_jobs" json:"open_jobs"`
	SpentCents int64        `db:"spent_cents" json:"spent_cents"`
	LastUsedOn *domain.Date `db:"last_used_on" json:"last_used_on"`
}

type VendorInput struct {
	Name        string `json:"name"`
	Trade       string `json:"trade"`
	ContactName string `json:"contact_name"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	Notes       string `json:"notes"`
}

const vendorSelect = `
	SELECT v.id, v.name, v.trade, v.contact_name, v.phone, v.email, v.notes, v.created_at,
	       (SELECT count(*) FROM maintenance_requests m
	         WHERE m.vendor_id = v.id AND m.status IN ('open', 'in_progress', 'on_hold'))::int AS open_jobs,
	       COALESCE((SELECT sum(e.amount_cents) FROM expenses e WHERE e.vendor_id = v.id), 0)::bigint AS spent_cents,
	       (SELECT max(e.spent_on) FROM expenses e WHERE e.vendor_id = v.id) AS last_used_on
	  FROM vendors v
	 WHERE v.owner_id = $1`

// ListVendors lists vendors by name, optionally only those of one trade.
func (s *Store) ListVendors(ctx context.Context, ownerID, trade string) ([]Vendor, error) {
	q := vendorSelect
	args := []any{ownerID}
	if trade != "" {
		q += ` AND v.trade = $2`
		args = append(args, trade)
	}
	rows, err := s.pool.Query(ctx, q+` ORDER BY lower(v.name)`, args...)
	return collectAll[Vendor](rows, err)
}

func (s *Store) GetVendor(ctx context.Context, ownerID, id string) (Vendor, error) {
	rows, err := s.pool.Query(ctx, vendorSelect+` AND v.id = $2`, ownerID, id)
	return collectOne[Vendor](rows, err)
}

const vendorExists = "a vendor with this name already exists"

func (s *Store) CreateVendor(ctx context.Context, ownerID string, in VendorInput) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO vendors (owner_id, name, trade, contact_name, phone, email, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		ownerID, in.Name, in.Trade, in.ContactName, in.Phone, in.Email, in.Notes,
	).Scan(&id)
	if isUniqueViolation(err) {
		return "", conflict(vendorExists)
	}
	return id, err
}

func (s *Store) UpdateVendor(ctx context.Context, ownerID, id string, in VendorInput) error {
	err := execOne(s.pool.Exec(ctx, `
		UPDATE vendors
		   SET name = $3, trade = $4, contact_name = $5, phone = $6, email = $7, notes = $8, updated_at = now()
		 WHERE owner_id = $1 AND id = $2`,
		ownerID, id, in.Name, in.Trade, in.ContactName, in.Phone, in.Email, in.Notes,
	))
	if isUniqueViolation(err) {
		return conflict(vendorExists)
	}
	return err
}

// DeleteVendor removes a vendor. Jobs and expenses keep the vendor's name
// as text but are no longer linked.
func (s *Store) DeleteVendor(ctx context.Context, ownerID, id string) error {
	return execOne(s.pool.Exec(ctx, `DELETE FROM vendors WHERE owner_id = $1 AND id = $2`, ownerID, id))
}

// checkVendor makes sure an optional vendor belongs to the owner.
func (s *Store) checkVendor(ctx context.Context, ownerID string, vendorID *string) error {
	if emptyToNil(vendorID) == nil {
		return nil
	}
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vendors WHERE id = $1 AND owner_id = $2)`, *vendorID, ownerID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return conflict("vendor not found")
	}
	return nil
}

// vendorNameAt is the SQL for the vendor text stored with a job or expense:
// the linked vendor's name if there is one, otherwise what was typed. text
// and id are the placeholders of the typed name and the vendor ID.
func vendorNameAt(text, id string) string {
	return `COALESCE((SELECT name FROM vendors WHERE id = ` + id + `::uuid), ` + text + `)`
}
