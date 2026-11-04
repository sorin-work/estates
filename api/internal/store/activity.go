package store

import (
	"context"
	"time"
)

// Activity is one entry in the change history.
type Activity struct {
	ID         int64     `db:"id" json:"id"`
	EntityType string    `db:"entity_type" json:"entity_type"`
	EntityID   string    `db:"entity_id" json:"entity_id"`
	LeaseID    *string   `db:"lease_id" json:"lease_id"`
	Action     string    `db:"action" json:"action"`
	Summary    string    `db:"summary" json:"summary"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type ActivityFilter struct {
	EntityType string
	EntityID   string
	LeaseID    string
	// Before is an activity ID; only older entries are returned.
	Before int64
	Limit  int
}

// RecordActivity adds an entry. CreatedAt defaults to now; the demo data
// sets it to back-date its history.
func (s *Store) RecordActivity(ctx context.Context, ownerID string, a Activity) error {
	var at *time.Time
	if !a.CreatedAt.IsZero() {
		at = &a.CreatedAt
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO activity (owner_id, entity_type, entity_id, lease_id, action, summary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, now()))`,
		ownerID, a.EntityType, a.EntityID, emptyToNil(a.LeaseID), a.Action, a.Summary, at)
	return err
}

// ListActivity returns the newest entries first.
func (s *Store) ListActivity(ctx context.Context, ownerID string, f ActivityFilter) ([]Activity, error) {
	w := newFilter(ownerID)
	w.raw("owner_id = $1")
	if f.EntityType != "" {
		w.add("entity_type = ?", f.EntityType)
	}
	if f.EntityID != "" {
		w.add("entity_id = ?", f.EntityID)
	}
	if f.LeaseID != "" {
		w.add("lease_id = ?", f.LeaseID)
	}
	if f.Before > 0 {
		w.add("id < ?", f.Before)
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, entity_type, entity_id, lease_id, action, summary, created_at
		  FROM activity`+w.where()+` ORDER BY id DESC LIMIT `+w.arg(limit), w.args...)
	return collectAll[Activity](rows, err)
}
