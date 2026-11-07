package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/sorin-work/estates/api/internal/domain"
)

type MaintenanceRequest struct {
	ID           string       `db:"id" json:"id"`
	PropertyID   string       `db:"property_id" json:"property_id"`
	PropertyName string       `db:"property_name" json:"property_name"`
	UnitID       *string      `db:"unit_id" json:"unit_id"`
	UnitLabel    *string      `db:"unit_label" json:"unit_label"`
	Title        string       `db:"title" json:"title"`
	Description  string       `db:"description" json:"description"`
	Priority     string       `db:"priority" json:"priority"`
	Status       string       `db:"status" json:"status"`
	Vendor       string       `db:"vendor" json:"vendor"`
	VendorID     *string      `db:"vendor_id" json:"vendor_id"`
	ReportedOn   domain.Date  `db:"reported_on" json:"reported_on"`
	CompletedOn  *domain.Date `db:"completed_on" json:"completed_on"`
	CostCents    int64        `db:"cost_cents" json:"cost_cents"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at" json:"updated_at"`
}

type MaintenanceInput struct {
	PropertyID  string      `json:"property_id"`
	UnitID      *string     `json:"unit_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Priority    string      `json:"priority"`
	Vendor      string      `json:"vendor"`
	VendorID    *string     `json:"vendor_id"`
	ReportedOn  domain.Date `json:"reported_on"`
}

type MaintenanceFilter struct {
	PropertyID string
	// Status is a single status, or "active" for anything not closed.
	Status   string
	Priority string
	VendorID string
	// Search matches the title, description and vendor.
	Search string
}

const maintenanceSelect = `
	SELECT m.id, m.property_id, p.name AS property_name, m.unit_id, u.label AS unit_label,
	       m.title, m.description, m.priority, m.status, COALESCE(vd.name, m.vendor) AS vendor, m.vendor_id,
	       m.reported_on, m.completed_on,
	       m.created_at, m.updated_at,
	       COALESCE((SELECT sum(e.amount_cents) FROM expenses e WHERE e.maintenance_request_id = m.id), 0)::bigint AS cost_cents
	  FROM maintenance_requests m
	  JOIN properties p ON p.id = m.property_id AND p.owner_id = $1
	  LEFT JOIN units u ON u.id = m.unit_id
	  LEFT JOIN vendors vd ON vd.id = m.vendor_id`

// priorityRank orders urgent first.
const priorityRank = `CASE m.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'normal' THEN 2 ELSE 3 END`

func (s *Store) ListMaintenance(ctx context.Context, ownerID string, f MaintenanceFilter) ([]MaintenanceRequest, error) {
	w := newFilter(ownerID)
	if f.PropertyID != "" {
		w.add("m.property_id = ?", f.PropertyID)
	}
	switch f.Status {
	case "":
	case "active":
		w.raw("m.status IN ('open', 'in_progress', 'on_hold')")
	default:
		w.add("m.status = ?", f.Status)
	}
	if f.Priority != "" {
		w.add("m.priority = ?", f.Priority)
	}
	if f.VendorID != "" {
		w.add("m.vendor_id = ?", f.VendorID)
	}
	if f.Search != "" {
		w.add("concat_ws(' ', m.title, m.description, m.vendor) ILIKE '%' || ? || '%'", escapeLike(f.Search))
	}
	q := maintenanceSelect + w.where() +
		` ORDER BY (m.status IN ('completed', 'cancelled')), ` + priorityRank + `, m.reported_on DESC, m.created_at DESC`
	rows, err := s.pool.Query(ctx, q, w.args...)
	return collectAll[MaintenanceRequest](rows, err)
}

func (s *Store) GetMaintenance(ctx context.Context, ownerID, id string) (MaintenanceRequest, error) {
	rows, err := s.pool.Query(ctx, maintenanceSelect+` WHERE m.id = $2`, ownerID, id)
	return collectOne[MaintenanceRequest](rows, err)
}

// checkUnitInProperty makes sure the optional unit belongs to the property.
func (s *Store) checkUnitInProperty(ctx context.Context, propertyID string, unitID *string) error {
	if unitID == nil || *unitID == "" {
		return nil
	}
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM units WHERE id = $1 AND property_id = $2)`, *unitID, propertyID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return conflict("the unit does not belong to this property")
	}
	return nil
}

func (s *Store) CreateMaintenance(ctx context.Context, ownerID string, in MaintenanceInput) (string, error) {
	if err := s.checkUnitInProperty(ctx, in.PropertyID, in.UnitID); err != nil {
		return "", err
	}
	if err := s.checkVendor(ctx, ownerID, in.VendorID); err != nil {
		return "", err
	}
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO maintenance_requests (property_id, unit_id, title, description, priority, vendor, vendor_id, reported_on)
		SELECT p.id, $3, $4, $5, $6, `+vendorNameAt("$7", "$8")+`, $8, $9 FROM properties p WHERE p.owner_id = $1 AND p.id = $2
		RETURNING id`,
		ownerID, in.PropertyID, emptyToNil(in.UnitID), in.Title, in.Description, in.Priority, in.Vendor,
		emptyToNil(in.VendorID), in.ReportedOn,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", conflict("property not found")
		}
		return "", err
	}
	return id, nil
}

func (s *Store) UpdateMaintenance(ctx context.Context, ownerID, id string, in MaintenanceInput) error {
	if err := s.checkUnitInProperty(ctx, in.PropertyID, in.UnitID); err != nil {
		return err
	}
	if err := s.checkVendor(ctx, ownerID, in.VendorID); err != nil {
		return err
	}
	return execOne(s.pool.Exec(ctx, `
		UPDATE maintenance_requests m
		   SET property_id = $3, unit_id = $4, title = $5, description = $6, priority = $7,
		       vendor = `+vendorNameAt("$8", "$9")+`, vendor_id = $9, reported_on = $10, updated_at = now()
		  FROM properties p, properties np
		 WHERE p.id = m.property_id AND p.owner_id = $1 AND m.id = $2
		   AND np.id = $3 AND np.owner_id = $1`,
		ownerID, id, in.PropertyID, emptyToNil(in.UnitID), in.Title, in.Description, in.Priority, in.Vendor,
		emptyToNil(in.VendorID), in.ReportedOn,
	))
}

// SetMaintenanceStatus moves a request to a new status. completedOn is set
// when the request is completed and cleared otherwise.
func (s *Store) SetMaintenanceStatus(ctx context.Context, ownerID, id, status string, completedOn *domain.Date) error {
	return execOne(s.pool.Exec(ctx, `
		UPDATE maintenance_requests m
		   SET status = $3, completed_on = $4, updated_at = now()
		  FROM properties p
		 WHERE p.id = m.property_id AND p.owner_id = $1 AND m.id = $2`,
		ownerID, id, status, completedOn))
}

func (s *Store) DeleteMaintenance(ctx context.Context, ownerID, id string) error {
	return execOne(s.pool.Exec(ctx, `
		DELETE FROM maintenance_requests m USING properties p
		 WHERE p.id = m.property_id AND p.owner_id = $1 AND m.id = $2`, ownerID, id))
}

func emptyToNil(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
