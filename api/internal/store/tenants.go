package store

import (
	"context"
	"strings"
	"time"
)

type Tenant struct {
	ID         string    `db:"id" json:"id"`
	FirstName  string    `db:"first_name" json:"first_name"`
	LastName   string    `db:"last_name" json:"last_name"`
	Email      string    `db:"email" json:"email"`
	Phone      string    `db:"phone" json:"phone"`
	Notes      string    `db:"notes" json:"notes"`
	LeaseCount int       `db:"lease_count" json:"lease_count"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type TenantInput struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Notes     string `json:"notes"`
}

const tenantSelect = `
	SELECT t.id, t.first_name, t.last_name, t.email, t.phone, t.notes, t.created_at,
	       (SELECT count(*) FROM leases l WHERE l.tenant_id = t.id)::int AS lease_count
	  FROM tenants t
	 WHERE t.owner_id = $1`

// ListTenants lists tenants, optionally filtered by a search over name,
// email and phone.
func (s *Store) ListTenants(ctx context.Context, ownerID, search string) ([]Tenant, error) {
	q := tenantSelect
	args := []any{ownerID}
	if search != "" {
		q += ` AND (t.first_name || ' ' || t.last_name || ' ' || t.email || ' ' || t.phone) ILIKE '%' || $2 || '%'`
		args = append(args, escapeLike(search))
	}
	rows, err := s.pool.Query(ctx, q+` ORDER BY lower(t.last_name), lower(t.first_name)`, args...)
	return collectAll[Tenant](rows, err)
}

func (s *Store) GetTenant(ctx context.Context, ownerID, id string) (Tenant, error) {
	rows, err := s.pool.Query(ctx, tenantSelect+` AND t.id = $2`, ownerID, id)
	return collectOne[Tenant](rows, err)
}

func (s *Store) CreateTenant(ctx context.Context, ownerID string, in TenantInput) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO tenants (owner_id, first_name, last_name, email, phone, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		ownerID, in.FirstName, in.LastName, in.Email, in.Phone, in.Notes,
	).Scan(&id)
	return id, err
}

func (s *Store) UpdateTenant(ctx context.Context, ownerID, id string, in TenantInput) error {
	return execOne(s.pool.Exec(ctx, `
		UPDATE tenants
		   SET first_name = $3, last_name = $4, email = $5, phone = $6, notes = $7, updated_at = now()
		 WHERE owner_id = $1 AND id = $2`,
		ownerID, id, in.FirstName, in.LastName, in.Email, in.Phone, in.Notes,
	))
}

func (s *Store) DeleteTenant(ctx context.Context, ownerID, id string) error {
	err := execOne(s.pool.Exec(ctx, `DELETE FROM tenants WHERE owner_id = $1 AND id = $2`, ownerID, id))
	if isForeignKeyViolation(err) {
		return conflict("this tenant has leases and cannot be deleted")
	}
	return err
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// escapeLike escapes the LIKE wildcards in user input, so searching for
// "50%" or "a_b" matches those characters literally.
func escapeLike(s string) string { return likeEscaper.Replace(s) }
