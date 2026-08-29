package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sorin-work/estates/api/internal/store"
)

type activityStore interface {
	RecordActivity(ctx context.Context, ownerID string, a store.Activity) error
	ListActivity(ctx context.Context, ownerID string, f store.ActivityFilter) ([]store.Activity, error)
}

// audit records a change in the history. The change itself has already
// happened, so a failure here is logged rather than failing the request.
func (s *Server) audit(r *http.Request, entityType, entityID, leaseID, action, summary string) {
	a := store.Activity{EntityType: entityType, EntityID: entityID, Action: action, Summary: summary}
	if leaseID != "" {
		a.LeaseID = &leaseID
	}
	if err := s.store.RecordActivity(r.Context(), userID(r), a); err != nil {
		s.log.Error("record activity", "entity_type", entityType, "entity_id", entityID, "err", err)
	}
}

// money formats cents for history entries, e.g. 145000 -> "1,450.00". The
// currency symbol is left to the reader.
func money(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := strconv.FormatInt(cents/100, 10)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	return fmt.Sprintf("%s%s.%02d", sign, whole, cents%100)
}

func leasePlace(l store.Lease) string {
	return l.PropertyName + " unit " + l.UnitLabel
}

func (s *Server) listActivity(w http.ResponseWriter, r *http.Request) error {
	var f store.ActivityFilter
	var err error
	if f.LeaseID, err = queryID(r, "lease_id"); err != nil {
		return err
	}
	if f.EntityID, err = queryID(r, "entity_id"); err != nil {
		return err
	}
	f.EntityType = r.URL.Query().Get("entity_type")
	before, err := queryInt(r, "before", 0)
	if err != nil {
		return err
	}
	f.Before = int64(before)
	if f.Limit, err = queryInt(r, "limit", 50); err != nil {
		return err
	}
	list, err := s.store.ListActivity(r.Context(), userID(r), f)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}
