package store

import (
	"context"
	"time"

	"github.com/sorin-work/estates/api/internal/domain"
)

type Property struct {
	ID            string    `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Kind          string    `db:"kind" json:"kind"`
	AddressLine1  string    `db:"address_line1" json:"address_line1"`
	AddressLine2  string    `db:"address_line2" json:"address_line2"`
	City          string    `db:"city" json:"city"`
	Region        string    `db:"region" json:"region"`
	PostalCode    string    `db:"postal_code" json:"postal_code"`
	Notes         string    `db:"notes" json:"notes"`
	UnitCount     int       `db:"unit_count" json:"unit_count"`
	OccupiedUnits int       `db:"occupied_units" json:"occupied_units"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type PropertyInput struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	Region       string `json:"region"`
	PostalCode   string `json:"postal_code"`
	Notes        string `json:"notes"`
}

// propertySelect expects $1 = owner ID and $2 = the date occupancy is
// measured on.
const propertySelect = `
	SELECT p.id, p.name, p.kind, p.address_line1, p.address_line2, p.city, p.region,
	       p.postal_code, p.notes, p.created_at, p.updated_at,
	       (SELECT count(*) FROM units u WHERE u.property_id = p.id)::int AS unit_count,
	       (SELECT count(DISTINCT u.id) FROM units u
	          JOIN leases l ON l.unit_id = u.id
	         WHERE u.property_id = p.id
	           AND l.start_date <= $2
	           AND LEAST(l.end_date, COALESCE(l.terminated_on, l.end_date)) >= $2)::int AS occupied_units
	  FROM properties p
	 WHERE p.owner_id = $1`

func (s *Store) ListProperties(ctx context.Context, ownerID string, asOf domain.Date) ([]Property, error) {
	rows, err := s.pool.Query(ctx, propertySelect+` ORDER BY lower(p.name)`, ownerID, asOf)
	return collectAll[Property](rows, err)
}

func (s *Store) GetProperty(ctx context.Context, ownerID, id string, asOf domain.Date) (Property, error) {
	rows, err := s.pool.Query(ctx, propertySelect+` AND p.id = $3`, ownerID, asOf, id)
	return collectOne[Property](rows, err)
}

func (s *Store) CreateProperty(ctx context.Context, ownerID string, in PropertyInput) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO properties (owner_id, name, kind, address_line1, address_line2, city, region, postal_code, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		ownerID, in.Name, in.Kind, in.AddressLine1, in.AddressLine2, in.City, in.Region, in.PostalCode, in.Notes,
	).Scan(&id)
	return id, err
}

func (s *Store) UpdateProperty(ctx context.Context, ownerID, id string, in PropertyInput) error {
	return execOne(s.pool.Exec(ctx, `
		UPDATE properties
		   SET name = $3, kind = $4, address_line1 = $5, address_line2 = $6, city = $7,
		       region = $8, postal_code = $9, notes = $10, updated_at = now()
		 WHERE owner_id = $1 AND id = $2`,
		ownerID, id, in.Name, in.Kind, in.AddressLine1, in.AddressLine2, in.City, in.Region, in.PostalCode, in.Notes,
	))
}

// DeleteProperty removes a property with its units. Properties with leases
// cannot be deleted, so rent history is never lost by accident.
func (s *Store) DeleteProperty(ctx context.Context, ownerID, id string) error {
	err := execOne(s.pool.Exec(ctx, `DELETE FROM properties WHERE owner_id = $1 AND id = $2`, ownerID, id))
	if isForeignKeyViolation(err) {
		return conflict("this property has leases; delete or move them first")
	}
	return err
}
