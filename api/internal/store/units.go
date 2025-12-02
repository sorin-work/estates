package store

import (
	"context"
	"time"

	"github.com/sorin-work/estates/api/internal/domain"
)

type Unit struct {
	ID              string    `db:"id" json:"id"`
	PropertyID      string    `db:"property_id" json:"property_id"`
	PropertyName    string    `db:"property_name" json:"property_name"`
	Label           string    `db:"label" json:"label"`
	Bedrooms        int       `db:"bedrooms" json:"bedrooms"`
	Bathrooms       float64   `db:"bathrooms" json:"bathrooms"`
	SquareFeet      *int      `db:"square_feet" json:"square_feet"`
	MarketRentCents int64     `db:"market_rent_cents" json:"market_rent_cents"`
	CurrentLeaseID  *string   `db:"current_lease_id" json:"current_lease_id"`
	CurrentTenant   *string   `db:"current_tenant" json:"current_tenant"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type UnitInput struct {
	Label           string  `json:"label"`
	Bedrooms        int     `json:"bedrooms"`
	Bathrooms       float64 `json:"bathrooms"`
	SquareFeet      *int    `json:"square_feet"`
	MarketRentCents int64   `json:"market_rent_cents"`
}

// unitSelect expects $1 = owner ID and $2 = the date occupancy is measured on.
const unitSelect = `
	SELECT u.id, u.property_id, p.name AS property_name, u.label, u.bedrooms, u.bathrooms,
	       u.square_feet, u.market_rent_cents, u.created_at,
	       cur.id AS current_lease_id, cur.tenant_name AS current_tenant
	  FROM units u
	  JOIN properties p ON p.id = u.property_id
	  LEFT JOIN LATERAL (
	        SELECT l.id, t.first_name || ' ' || t.last_name AS tenant_name
	          FROM leases l
	          JOIN tenants t ON t.id = l.tenant_id
	         WHERE l.unit_id = u.id
	           AND l.start_date <= $2
	           AND LEAST(l.end_date, COALESCE(l.terminated_on, l.end_date)) >= $2
	         ORDER BY l.start_date DESC
	         LIMIT 1
	  ) cur ON true
	 WHERE p.owner_id = $1`

// ListUnits lists units, optionally only those of one property.
func (s *Store) ListUnits(ctx context.Context, ownerID, propertyID string, asOf domain.Date) ([]Unit, error) {
	q := unitSelect
	args := []any{ownerID, asOf}
	if propertyID != "" {
		q += ` AND u.property_id = $3`
		args = append(args, propertyID)
	}
	rows, err := s.pool.Query(ctx, q+` ORDER BY lower(p.name), u.label`, args...)
	return collectAll[Unit](rows, err)
}

func (s *Store) GetUnit(ctx context.Context, ownerID, id string, asOf domain.Date) (Unit, error) {
	rows, err := s.pool.Query(ctx, unitSelect+` AND u.id = $3`, ownerID, asOf, id)
	return collectOne[Unit](rows, err)
}

func (s *Store) CreateUnit(ctx context.Context, ownerID, propertyID string, in UnitInput) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO units (property_id, label, bedrooms, bathrooms, square_feet, market_rent_cents)
		SELECT p.id, $3, $4, $5, $6, $7 FROM properties p WHERE p.owner_id = $1 AND p.id = $2
		RETURNING id`,
		ownerID, propertyID, in.Label, in.Bedrooms, in.Bathrooms, in.SquareFeet, in.MarketRentCents,
	).Scan(&id)
	if isUniqueViolation(err) {
		return "", conflict("a unit with this label already exists in the property")
	}
	return id, notFound(err)
}

func (s *Store) UpdateUnit(ctx context.Context, ownerID, id string, in UnitInput) error {
	err := execOne(s.pool.Exec(ctx, `
		UPDATE units u
		   SET label = $3, bedrooms = $4, bathrooms = $5, square_feet = $6,
		       market_rent_cents = $7, updated_at = now()
		  FROM properties p
		 WHERE p.id = u.property_id AND p.owner_id = $1 AND u.id = $2`,
		ownerID, id, in.Label, in.Bedrooms, in.Bathrooms, in.SquareFeet, in.MarketRentCents,
	))
	if isUniqueViolation(err) {
		return conflict("a unit with this label already exists in the property")
	}
	return err
}

func (s *Store) DeleteUnit(ctx context.Context, ownerID, id string) error {
	err := execOne(s.pool.Exec(ctx, `
		DELETE FROM units u USING properties p
		 WHERE p.id = u.property_id AND p.owner_id = $1 AND u.id = $2`, ownerID, id))
	if isForeignKeyViolation(err) {
		return conflict("this unit has leases and cannot be deleted")
	}
	return err
}
