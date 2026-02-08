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
