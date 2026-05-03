package httpapi

import (
	"context"
	"net/http"

	"github.com/sorin-work/estates/api/internal/billing"
	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type dashboardStore interface {
	Occupancy(ctx context.Context, ownerID string, asOf domain.Date) (store.Occupancy, error)
	MonthlyTotals(ctx context.Context, ownerID string, from, to domain.Date, propertyID string) ([]store.MonthTotal, error)
	ExpiringLeases(ctx context.Context, ownerID string, asOf domain.Date, days int) ([]store.Lease, error)
	MaintenanceCounts(ctx context.Context, ownerID string) (map[string]int, error)
}

type dashboardResponse struct {
	AsOf      domain.Date `json:"as_of"`
	Occupancy struct {
		Units    int     `json:"units"`
		Occupied int     `json:"occupied"`
		Vacant   int     `json:"vacant"`
		Rate     float64 `json:"rate"`
	} `json:"occupancy"`
	ThisMonth   store.MonthTotal `json:"this_month"`
	Outstanding struct {
		BalanceCents  int64 `json:"balance_cents"`
		OverdueCents  int64 `json:"overdue_cents"`
		OverdueLeases int   `json:"overdue_leases"`
	} `json:"outstanding"`
	// Deposits is the money held for tenants. It is a liability, not income.
	Deposits struct {
		HeldCents int64 `json:"held_cents"`
		Leases    int   `json:"leases"`
	} `json:"deposits"`
	Overdue        []billing.Balance `json:"overdue"`
	ExpiringLeases []store.Lease     `json:"expiring_leases"`
	Maintenance    struct {
		Open       int               `json:"open"`
		ByPriority map[string]int    `json:"by_priority"`
		Recent     []maintenanceView `json:"recent"`
	} `json:"maintenance"`
	Monthly []store.MonthTotal `json:"monthly"`
}

// dashboard returns the numbers for the home screen. The 12 month chart ends
// with the current month.
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	owner := userID(r)
	today := s.today()

	var resp dashboardResponse
	resp.AsOf = today

	occ, err := s.store.Occupancy(ctx, owner, today)
	if err != nil {
		return err
	}
	resp.Occupancy.Units = occ.Units
	resp.Occupancy.Occupied = occ.Occupied
	resp.Occupancy.Vacant = occ.Units - occ.Occupied
	if occ.Units > 0 {
		resp.Occupancy.Rate = float64(occ.Occupied) / float64(occ.Units)
	}

	from := today.MonthStart().AddMonths(-11)
	monthly, err := s.store.MonthlyTotals(ctx, owner, from, today, "")
	if err != nil {
		return err
	}
	resp.Monthly = monthly
	if len(monthly) > 0 {
		resp.ThisMonth = monthly[len(monthly)-1]
	}

	balances, err := s.billing.Balances(ctx, owner, today)
	if err != nil {
		return err
	}
	resp.Overdue = []billing.Balance{}
	for _, b := range balances {
		if b.BalanceCents > 0 {
			resp.Outstanding.BalanceCents += b.BalanceCents
		}
		if b.OverdueCents > 0 {
			resp.Outstanding.OverdueCents += b.OverdueCents
			resp.Outstanding.OverdueLeases++
			if len(resp.Overdue) < 5 {
				resp.Overdue = append(resp.Overdue, b)
			}
		}
	}

	held, err := s.store.DepositsHeld(ctx, owner)
	if err != nil {
		return err
	}
	for _, cents := range held {
		resp.Deposits.HeldCents += cents
		resp.Deposits.Leases++
	}

	expiring, err := s.store.ExpiringLeases(ctx, owner, today, domain.ExpiringWindowDays)
	if err != nil {
		return err
	}
	for i := range expiring {
		expiring[i] = s.withStatus(expiring[i])
	}
	resp.ExpiringLeases = expiring

	counts, err := s.store.MaintenanceCounts(ctx, owner)
	if err != nil {
		return err
	}
	resp.Maintenance.ByPriority = counts
	for _, n := range counts {
		resp.Maintenance.Open += n
	}
	recent, err := s.store.ListMaintenance(ctx, owner, store.MaintenanceFilter{Status: "active"})
	if err != nil {
		return err
	}
	resp.Maintenance.Recent = []maintenanceView{}
	for _, m := range recent[:min(len(recent), 5)] {
		resp.Maintenance.Recent = append(resp.Maintenance.Recent, viewMaintenance(m))
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, resp)
	return nil
}
