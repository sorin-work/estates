package seed

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/sorin-work/estates/api/internal/billing"
	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
	"github.com/sorin-work/estates/api/internal/testdb"
)

func TestDemoSeed(t *testing.T) {
	ctx := context.Background()
	st := store.New(testdb.New(t))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bill := billing.New(st, log)

	if err := Demo(ctx, st, bill, log); err != nil {
		t.Fatal(err)
	}
	// Running it again is a no-op.
	if err := Demo(ctx, st, bill, log); err != nil {
		t.Fatal(err)
	}

	u, err := st.UserByEmail(ctx, DemoEmail)
	if err != nil {
		t.Fatal(err)
	}
	today := domain.Today()

	occ, err := st.Occupancy(ctx, u.ID, today)
	if err != nil {
		t.Fatal(err)
	}
	if occ.Units != 10 || occ.Occupied != 9 {
		t.Errorf("occupancy %+v", occ)
	}

	balances, err := bill.Balances(ctx, u.ID, today)
	if err != nil {
		t.Fatal(err)
	}
	overdue := 0
	for _, b := range balances {
		if b.OverdueCents > 0 {
			overdue++
		}
	}
	if overdue == 0 {
		t.Error("expected at least one overdue lease in the demo data")
	}

	months, err := st.MonthlyTotals(ctx, u.ID, today.MonthStart().AddMonths(-11), today, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range months[:11] {
		if m.CollectedCents == 0 || m.ExpensesCents == 0 {
			t.Errorf("month %s has no activity: %+v", m.Month, m)
		}
	}

	leases, err := st.ListLeases(ctx, u.ID, store.LeaseFilter{})
	if err != nil {
		t.Fatal(err)
	}
	var increased, renewed bool
	for _, l := range leases {
		increased = increased || len(l.RentChanges) > 0
		renewed = renewed || l.RenewedFromID != nil
	}
	if !increased || !renewed {
		t.Errorf("expected a rent increase (%v) and a renewal (%v)", increased, renewed)
	}

	history, err := st.ListActivity(ctx, u.ID, store.ActivityFilter{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	if len(history) < 20 {
		t.Errorf("only %d activity entries", len(history))
	}
	for i := 1; i < len(history); i++ {
		if history[i].CreatedAt.After(history[i-1].CreatedAt) {
			t.Fatalf("activity is not newest first at %d: %v after %v", i, history[i].CreatedAt, history[i-1].CreatedAt)
		}
	}

	recurring, err := st.ListRecurringExpenses(ctx, u.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(recurring) != 4 {
		t.Errorf("%d recurring expenses", len(recurring))
	}
	for _, r := range recurring {
		if r.GeneratedThrough == nil {
			t.Errorf("recurring %s was never generated", r.Description)
		}
	}

	vendors, err := st.ListVendors(ctx, u.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(vendors) != len(demoVendors) {
		t.Errorf("%d vendors", len(vendors))
	}
	for _, v := range vendors {
		if v.SpentCents == 0 {
			t.Errorf("vendor %s has no expenses", v.Name)
		}
	}

	active, err := st.ListMaintenance(ctx, u.ID, store.MaintenanceFilter{Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	if len(active) < 3 {
		t.Errorf("only %d open maintenance requests", len(active))
	}
}
