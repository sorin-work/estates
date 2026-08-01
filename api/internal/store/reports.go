package store

import (
	"context"

	"github.com/sorin-work/estates/api/internal/domain"
)

// PropertyResult is one property's income and costs over a period.
type PropertyResult struct {
	PropertyID   string `db:"property_id" json:"property_id"`
	PropertyName string `db:"property_name" json:"property_name"`
	// ChargedCents is rent, fees and other charges that fell due.
	ChargedCents int64 `db:"charged_cents" json:"charged_cents"`
	// CollectedCents is payments received.
	CollectedCents int64 `db:"collected_cents" json:"collected_cents"`
	// WithheldCents is deposit money kept to cover damage or unpaid rent.
	WithheldCents int64           `db:"withheld_cents" json:"withheld_cents"`
	ExpensesCents int64           `db:"expenses_cents" json:"expenses_cents"`
	ByCategory    []CategoryTotal `db:"-" json:"by_category"`
}

// NetCents is what the property earned: money in less money out.
func (p PropertyResult) NetCents() int64 {
	return p.CollectedCents + p.WithheldCents - p.ExpensesCents
}

// PropertyResults returns income and expenses for each of the owner's
// properties between from and to, inclusive, by property name.
func (s *Store) PropertyResults(ctx context.Context, ownerID string, from, to domain.Date) ([]PropertyResult, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id AS property_id, p.name AS property_name,
		       COALESCE((SELECT sum(c.amount_cents) FROM charges c
		                   JOIN leases l ON l.id = c.lease_id JOIN units u ON u.id = l.unit_id
		                  WHERE u.property_id = p.id AND c.kind <> 'deposit'
		                    AND c.due_date BETWEEN $2 AND $3), 0)::bigint AS charged_cents,
		       COALESCE((SELECT sum(pm.amount_cents) FROM payments pm
		                   JOIN leases l ON l.id = pm.lease_id JOIN units u ON u.id = l.unit_id
		                  WHERE u.property_id = p.id AND pm.paid_on BETWEEN $2 AND $3), 0)::bigint AS collected_cents,
		       COALESCE((SELECT sum(d.amount_cents) FROM deposit_transactions d
		                   JOIN leases l ON l.id = d.lease_id JOIN units u ON u.id = l.unit_id
		                  WHERE u.property_id = p.id AND d.kind = 'withheld'
		                    AND d.occurred_on BETWEEN $2 AND $3), 0)::bigint AS withheld_cents,
		       COALESCE((SELECT sum(e.amount_cents) FROM expenses e
		                  WHERE e.property_id = p.id AND e.spent_on BETWEEN $2 AND $3), 0)::bigint AS expenses_cents
		  FROM properties p
		 WHERE p.owner_id = $1
		 ORDER BY lower(p.name)`, ownerID, from, to)
	results, err := collectAll[PropertyResult](rows, err)
	if err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `
		SELECT e.property_id::text, e.category, sum(e.amount_cents)::bigint
		  FROM expenses e
		  JOIN properties p ON p.id = e.property_id AND p.owner_id = $1
		 WHERE e.spent_on BETWEEN $2 AND $3
		 GROUP BY e.property_id, e.category
		 ORDER BY 3 DESC`, ownerID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byProperty := map[string][]CategoryTotal{}
	for rows.Next() {
		var id string
		var ct CategoryTotal
		if err := rows.Scan(&id, &ct.Category, &ct.AmountCents); err != nil {
			return nil, err
		}
		byProperty[id] = append(byProperty[id], ct)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range results {
		results[i].ByCategory = byProperty[results[i].PropertyID]
		if results[i].ByCategory == nil {
			results[i].ByCategory = []CategoryTotal{}
		}
	}
	return results, nil
}
