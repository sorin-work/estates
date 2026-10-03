package httpapi

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/sorin-work/estates/api/internal/store"
)

type searchStore interface {
	Search(ctx context.Context, ownerID, q string) ([]store.SearchHit, error)
}

// search is the quick search in the sidebar. Very short queries match too
// much to be useful, so they return nothing.
func (s *Server) search(w http.ResponseWriter, r *http.Request) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(q) < 2 {
		writeJSON(w, http.StatusOK, []store.SearchHit{})
		return nil
	}
	if utf8.RuneCountInString(q) > 100 {
		return errBadRequest("search is too long")
	}
	hits, err := s.store.Search(r.Context(), userID(r), q)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, hits)
	return nil
}
