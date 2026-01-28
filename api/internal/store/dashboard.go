package store

import (
	"context"

	"github.com/sorin-work/estates/api/internal/domain"
)

// Occupancy counts units and how many have a lease in force on a day.
type Occupancy struct {
	Units    int `json:"units"`
	Occupied int `json:"occupied"`
}

func (s *Store) Occupancy(ctx context.Context, ownerID string, asOf domain.Date) (Occupancy, error) {
	var o Occupancy
	err := s.pool.QueryRow(ctx, `
		SELECT count(*)::int,
		       count(*) FILTER (WHERE EXISTS (
		           SELECT 1 FROM leases l
		            WHERE l.unit_id = u.id AND l.start_date <= $2
		              AND LEAST(l.end_date, COALESCE(l.terminated_on, l.end_date)) >= $2))::int
		  FROM units u
		  JOIN properties p ON p.id = u.property_id AND p.owner_id = $1`, ownerID, asOf,
	).Scan(&o.Units, &o.Occupied)
	return o, err
}

// MonthTotal is income and expenses for one calendar month.
type MonthTotal struct {
	Month          string `db:"month" json:"month"`
	ChargedCents   int64  `db:"charged_cents" json:"charged_cents"`
	CollectedCents int64  `db:"collected_cents" json:"collected_cents"`
	ExpensesCents  int64  `db:"expenses_cents" json:"expenses_cents"`
}

// MonthlyTotals returns one row per month from from's month to to's month,
// including months with no activity. Charges count rent and fees falling due
// in the month, collected counts payments received in it.
func (s *Store) MonthlyTotals(ctx context.Context, ownerID string, from, to domain.Date, propertyID string) ([]MonthTotal, error) {
	rows, err := s.pool.Query(ctx, `
		WITH months AS (
		    SELECT generate_series(date_trunc('month', $2::date), date_trunc('month', $3::date), interval '1 month')::date AS m
		), owned AS (
		    SELECT l.id FROM leases l
		      JOIN units u ON u.id = l.unit_id
		      JOIN properties p ON p.id = u.property_id AND p.owner_id = $1
		     WHERE $4 = '' OR p.id::text = $4
		)
		SELECT to_char(months.m, 'YYYY-MM') AS month,
		       COALESCE((SELECT sum(c.amount_cents) FROM charges c
		                  WHERE c.lease_id IN (SELECT id FROM owned)
		                    AND c.kind <> 'deposit'
		                    AND date_trunc('month', c.due_date) = months.m), 0)::bigint AS charged_cents,
		       COALESCE((SELECT sum(pm.amount_cents) FROM payments pm
		                  WHERE pm.lease_id IN (SELECT id FROM owned)
		                    AND date_trunc('month', pm.paid_on) = months.m), 0)::bigint AS collected_cents,
		       COALESCE((SELECT sum(e.amount_cents) FROM expenses e
		                   JOIN properties p ON p.id = e.property_id AND p.owner_id = $1
		                  WHERE ($4 = '' OR p.id::text = $4)
		                    AND date_trunc('month', e.spent_on) = months.m), 0)::bigint AS expenses_cents
		  FROM months
		 ORDER BY months.m`, ownerID, from, to, propertyID)
	return collectAll[MonthTotal](rows, err)
}

// ExpiringLeases returns leases in force on asOf whose last day falls within
// the next `days` days, soonest first.
func (s *Store) ExpiringLeases(ctx context.Context, ownerID string, asOf domain.Date, days int) ([]Lease, error) {
	rows, err := s.pool.Query(ctx, leaseSelect+`
		WHERE p.owner_id = $1
		  AND l.start_date <= $2
		  AND LEAST(l.end_date, COALESCE(l.terminated_on, l.end_date)) BETWEEN $2 AND $2::date + $3::int
		ORDER BY LEAST(l.end_date, COALESCE(l.terminated_on, l.end_date))`, ownerID, asOf, days)
	return collectAll[Lease](rows, err)
}

// MaintenanceCounts counts requests that still need attention, by priority.
func (s *Store) MaintenanceCounts(ctx context.Context, ownerID string) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.priority, count(*)::int
		  FROM maintenance_requests m
		  JOIN properties p ON p.id = m.property_id AND p.owner_id = $1
		 WHERE m.status IN ('open', 'in_progress', 'on_hold')
		 GROUP BY m.priority`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for _, p := range domain.MaintenancePriorities {
		out[p] = 0
	}
	for rows.Next() {
		var p string
		var n int
		if err := rows.Scan(&p, &n); err != nil {
			return nil, err
		}
		out[p] = n
	}
	return out, rows.Err()
}
