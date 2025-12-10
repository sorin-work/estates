// Package billing turns lease terms into rent charges and works out lease
// balances for the rent views.
package billing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

// Store is the subset of the data layer billing needs.
type Store interface {
	BillableLeases(ctx context.Context, asOf domain.Date) ([]store.Lease, error)
	InsertRent(ctx context.Context, leaseID string, rent []domain.ScheduledRent) (int, error)
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
