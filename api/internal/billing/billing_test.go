package billing_test

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

func TestBillAllAcrossOwners(t *testing.T) {
	ctx := context.Background()
	st := store.New(testdb.New(t))
	svc := billing.New(st, slog.New(slog.NewTextHandler(io.Discard, nil)))

	for _, email := range []string{"a@example.com", "b@example.com"} {
		u, _ := st.CreateUser(ctx, email, "Owner", "hash")
		p, _ := st.CreateProperty(ctx, u.ID, store.PropertyInput{Name: "P", Kind: "condo", AddressLine1: "1 Main", City: "Bend"})
		unit, _ := st.CreateUnit(ctx, u.ID, p, store.UnitInput{Label: "1"})
		tenant, _ := st.CreateTenant(ctx, u.ID, store.TenantInput{FirstName: "T", LastName: "T"})
		_, err := st.CreateLease(ctx, u.ID, store.LeaseInput{
			UnitID: unit, TenantID: tenant, StartDate: domain.MustParseDate("2026-01-01"), EndDate: domain.MustParseDate("2026-12-31"),
			RentCents: 100000, DueDay: 1, GraceDays: 2, LateFeeCents: 2500,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	res, err := svc.BillAll(ctx, domain.MustParseDate("2026-02-05"))
	if err != nil {
		t.Fatal(err)
	}
	// Two leases, January and February rent each. Both months are unpaid,
	// but January became overdue more than 30 days ago, so only February
	// gets a late fee.
	if res.Leases != 2 || res.Rent != 4 || res.LateFees != 2 {
		t.Errorf("result %+v", res)
	}

	res, _ = svc.BillAll(ctx, domain.MustParseDate("2026-02-05"))
	if res.Rent != 0 || res.LateFees != 0 {
		t.Errorf("rerun added charges: %+v", res)
	}

	// A lease that ended long ago is not billed any more.
	res, _ = svc.BillAll(ctx, domain.MustParseDate("2027-06-01"))
	if res.Leases != 0 {
		t.Errorf("billed %d ended leases", res.Leases)
	}
}

func TestRecurringExpenses(t *testing.T) {
	ctx := context.Background()
	st := store.New(testdb.New(t))
	svc := billing.New(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
	d := domain.MustParseDate

	u, _ := st.CreateUser(ctx, "a@example.com", "Owner", "hash")
	p, _ := st.CreateProperty(ctx, u.ID, store.PropertyInput{Name: "P", Kind: "condo", AddressLine1: "1 Main", City: "Bend"})
	id, err := st.CreateRecurringExpense(ctx, u.ID, store.RecurringExpenseInput{
		PropertyID: p, Vendor: "First Coastal Bank", Category: "mortgage", AmountCents: 142000,
		Description: "Mortgage", DayOfMonth: 1, EveryMonths: 1, StartDate: d("2026-01-01"),
	})
	if err != nil {
		t.Fatal(err)
	}

	n, err := svc.GenerateAllRecurring(ctx, d("2026-03-10"))
	if err != nil || n != 3 {
		t.Fatalf("first run: %d %v", n, err)
	}
	if n, _ := svc.GenerateAllRecurring(ctx, d("2026-03-10")); n != 0 {
		t.Errorf("rerun added %d", n)
	}

	// An occurrence deleted by hand is not created again.
	list, _ := st.ListExpenses(ctx, u.ID, store.ExpenseFilter{})
	if len(list) != 3 || list[0].RecurringExpenseID == nil || *list[0].RecurringExpenseID != id {
		t.Fatalf("expenses %+v", list)
	}
	if err := st.DeleteExpense(ctx, u.ID, list[0].ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := svc.GenerateAllRecurring(ctx, d("2026-04-02")); n != 1 {
		t.Errorf("april run added %d, want only april", n)
	}
	list, _ = st.ListExpenses(ctx, u.ID, store.ExpenseFilter{})
	if len(list) != 3 || list[0].SpentOn.String() != "2026-04-01" {
		t.Errorf("after april %+v", list)
	}
}

func TestBalancesShowTheRentInForce(t *testing.T) {
	ctx := context.Background()
	st := store.New(testdb.New(t))
	svc := billing.New(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
	d := domain.MustParseDate

	u, _ := st.CreateUser(ctx, "a@example.com", "Owner", "hash")
	p, _ := st.CreateProperty(ctx, u.ID, store.PropertyInput{Name: "P", Kind: "condo", AddressLine1: "1 Main", City: "Bend"})
	unit, _ := st.CreateUnit(ctx, u.ID, p, store.UnitInput{Label: "1"})
	tenant, _ := st.CreateTenant(ctx, u.ID, store.TenantInput{FirstName: "T", LastName: "T"})
	id, err := st.CreateLease(ctx, u.ID, store.LeaseInput{
		UnitID: unit, TenantID: tenant, StartDate: d("2026-01-01"), EndDate: d("2026-12-31"), RentCents: 100000, DueDay: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddRentChange(ctx, u.ID, id, store.RentChangeInput{EffectiveDate: d("2026-06-01"), RentCents: 103000}); err != nil {
		t.Fatal(err)
	}

	balances, err := svc.Balances(ctx, u.ID, d("2026-07-10"))
	if err != nil {
		t.Fatal(err)
	}
	if len(balances) != 1 || balances[0].Lease.CurrentRentCents != 103000 {
		t.Errorf("balances %+v", balances)
	}
}
