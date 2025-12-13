package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/billing"
)

func (s *Server) rentRoutes(r chi.Router) {
	r.Get("/balances", s.handle(s.rentBalances))
	r.Post("/run", s.handle(s.runBilling))
}

// rentBalances is the rent roll: every active lease, plus ended ones that
// still owe money. ?overdue=true limits it to leases with overdue rent.
func (s *Server) rentBalances(w http.ResponseWriter, r *http.Request) error {
	balances, err := s.billing.Balances(r.Context(), userID(r), s.today())
	if err != nil {
		return err
	}
	if r.URL.Query().Get("overdue") == "true" {
		filtered := []billing.Balance{}
		for _, b := range balances {
			if b.OverdueCents > 0 {
				filtered = append(filtered, b)
			}
		}
		balances = filtered
	}
	writeJSON(w, http.StatusOK, balances)
	return nil
}

func (s *Server) runBilling(w http.ResponseWriter, r *http.Request) error {
	res, err := s.billing.BillOwner(r.Context(), userID(r), s.today())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, res)
	return nil
}
