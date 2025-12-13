// Package billing turns lease terms into rent charges and works out lease
// balances for the rent views.
package billing

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

// Store is the subset of the data layer billing needs.
type Store interface {
	BillableLeases(ctx context.Context, asOf domain.Date) ([]store.Lease, error)
	ListLeases(ctx context.Context, ownerID string, f store.LeaseFilter) ([]store.Lease, error)
	InsertRent(ctx context.Context, leaseID string, rent []domain.ScheduledRent) (int, error)
	LedgerEntries(ctx context.Context, ownerID, leaseID string) (map[string]*store.LedgerEntries, error)
}

type Service struct {
	store Store
	log   *slog.Logger
}

func New(s Store, log *slog.Logger) *Service {
	return &Service{store: s, log: log}
}

// Result counts what a billing run added.
type Result struct {
	Leases   int `json:"leases"`
	Rent     int `json:"rent_charges"`
	LateFees int `json:"late_fees"`
}

// GenerateRent adds any missing rent charges for the lease up to asOf. It
// does not add late fees; those are left to the scheduled run so that
// entering a lease does not immediately penalise the tenant.
func (s *Service) GenerateRent(ctx context.Context, l store.Lease, asOf domain.Date) (int, error) {
	added, err := s.store.InsertRent(ctx, l.ID, domain.RentSchedule(l.Terms(), asOf))
	if err != nil {
		return 0, fmt.Errorf("insert rent for lease %s: %w", l.ID, err)
	}
	return added, nil
}

// BillLease generates any missing rent charges for the lease up to asOf.
func (s *Service) BillLease(ctx context.Context, l store.Lease, asOf domain.Date) (Result, error) {
	res := Result{Leases: 1}
	added, err := s.GenerateRent(ctx, l, asOf)
	if err != nil {
		return res, err
	}
	res.Rent = added

	return res, nil
}

// BillAll runs BillLease for every lease that may need charges. A failure on
// one lease is logged and does not stop the others.
func (s *Service) BillAll(ctx context.Context, asOf domain.Date) (Result, error) {
	leases, err := s.store.BillableLeases(ctx, asOf)
	if err != nil {
		return Result{}, err
	}
	var total Result
	for _, l := range leases {
		r, err := s.BillLease(ctx, l, asOf)
		if err != nil {
			s.log.Error("billing failed", "lease_id", l.ID, "err", err)
			continue
		}
		total.Leases++
		total.Rent += r.Rent
		total.LateFees += r.LateFees
	}
	return total, nil
}

// BillOwner bills every lease of one owner. It backs the "generate charges
// now" button so landlords do not have to wait for the next scheduled run.
func (s *Service) BillOwner(ctx context.Context, ownerID string, asOf domain.Date) (Result, error) {
	leases, err := s.store.ListLeases(ctx, ownerID, store.LeaseFilter{})
	if err != nil {
		return Result{}, err
	}
	var total Result
	for _, l := range leases {
		if l.StartDate.After(asOf) {
			continue
		}
		r, err := s.BillLease(ctx, l, asOf)
		if err != nil {
			return total, err
		}
		total.Leases++
		total.Rent += r.Rent
		total.LateFees += r.LateFees
	}
	return total, nil
}

// Run bills all leases now and then on every tick until ctx is cancelled.
func (s *Service) Run(ctx context.Context, every time.Duration) {
	run := func() {
		res, err := s.BillAll(ctx, domain.Today())
		if err != nil {
			s.log.Error("billing run failed", "err", err)
			return
		}
		s.log.Info("billing run finished", "leases", res.Leases, "rent_charges", res.Rent, "late_fees", res.LateFees)
	}
	run()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

// Balance is one lease's position as shown in the rent roll.
type Balance struct {
	Lease         store.Lease  `json:"lease"`
	BalanceCents  int64        `json:"balance_cents"`
	OverdueCents  int64        `json:"overdue_cents"`
	OldestOverdue *domain.Date `json:"oldest_overdue"`
	DaysOverdue   int          `json:"days_overdue"`
}

// Balances returns the balance of every lease that is in force or still has
// money owing, sorted with the most overdue first.
func (s *Service) Balances(ctx context.Context, ownerID string, asOf domain.Date) ([]Balance, error) {
	leases, err := s.store.ListLeases(ctx, ownerID, store.LeaseFilter{})
	if err != nil {
		return nil, err
	}
	entries, err := s.store.LedgerEntries(ctx, ownerID, "")
	if err != nil {
		return nil, err
	}

	out := []Balance{}
	for _, l := range leases {
		l.Status = domain.LeaseStatus(l.StartDate, l.EndDate, l.TerminatedOn, asOf)
		e := entries[l.ID]
		if e == nil {
			e = &store.LedgerEntries{}
		}
		ledger := domain.BuildLedger(e.Charges, e.Payments, l.GraceDays, asOf)
		if l.Status != domain.LeaseActive && ledger.BalanceCents == 0 {
			continue
		}
		b := Balance{
			Lease:        l,
			BalanceCents: ledger.BalanceCents,
			OverdueCents: ledger.OverdueCents,
			DaysOverdue:  ledger.DaysOverdue,
		}
		if !ledger.OldestOverdue.IsZero() {
			d := ledger.OldestOverdue
			b.OldestOverdue = &d
		}
		out = append(out, b)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DaysOverdue != out[j].DaysOverdue {
			return out[i].DaysOverdue > out[j].DaysOverdue
		}
		return out[i].BalanceCents > out[j].BalanceCents
	})
	return out, nil
}
