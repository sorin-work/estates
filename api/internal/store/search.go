package store

import (
	"context"
)

// SearchHit is one match in the quick search.
type SearchHit struct {
	Kind  string `db:"kind" json:"kind"`
	ID    string `db:"id" json:"id"`
	Title string `db:"title" json:"title"`
	// Detail says more about the match, e.g. an address or email.
	Detail string `db:"detail" json:"detail"`
	// PropertyID is set for units, which have no page of their own.
	PropertyID string `db:"property_id" json:"property_id,omitempty"`
}

// searchLimit is how many hits of each kind are returned.
const searchLimit = 5

// Search finds properties, units, tenants and vendors whose name or
// details contain q, ignoring case. Names that start with q come first.
func (s *Store) Search(ctx context.Context, ownerID, q string) ([]SearchHit, error) {
	pattern := "%" + escapeLike(q) + "%"
	prefix := escapeLike(q) + "%"
	rows, err := s.pool.Query(ctx, `
		(SELECT 'property' AS kind, p.id::text AS id, p.name AS title,
		        concat_ws(', ', p.address_line1, p.city) AS detail, '' AS property_id
		   FROM properties p
		  WHERE p.owner_id = $1
		    AND concat_ws(' ', p.name, p.address_line1, p.address_line2, p.city, p.postal_code) ILIKE $2
		  ORDER BY p.name ILIKE $3 DESC, lower(p.name) LIMIT $4)
		UNION ALL
		(SELECT 'unit', u.id::text, p.name || ' · ' || u.label, 'Unit', p.id::text
		   FROM units u JOIN properties p ON p.id = u.property_id
		  WHERE p.owner_id = $1 AND (p.name || ' ' || u.label) ILIKE $2
		  ORDER BY u.label ILIKE $3 DESC, lower(p.name), u.label LIMIT $4)
		UNION ALL
		(SELECT 'tenant', t.id::text, t.first_name || ' ' || t.last_name, concat_ws(' · ', NULLIF(t.email, ''), NULLIF(t.phone, '')), ''
		   FROM tenants t
		  WHERE t.owner_id = $1
		    AND concat_ws(' ', t.first_name, t.last_name, t.email, t.phone) ILIKE $2
		  ORDER BY (t.first_name ILIKE $3 OR t.last_name ILIKE $3) DESC, lower(t.last_name), lower(t.first_name) LIMIT $4)
		UNION ALL
		(SELECT 'vendor', v.id::text, v.name, v.trade, ''
		   FROM vendors v
		  WHERE v.owner_id = $1 AND concat_ws(' ', v.name, v.contact_name, v.email, v.phone) ILIKE $2
		  ORDER BY v.name ILIKE $3 DESC, lower(v.name) LIMIT $4)`,
		ownerID, pattern, prefix, searchLimit)
	return collectAll[SearchHit](rows, err)
}
