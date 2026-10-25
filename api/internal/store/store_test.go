package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
	"github.com/sorin-work/estates/api/internal/testdb"
)

var ctx = context.Background()

func d(s string) domain.Date { return domain.MustParseDate(s) }

type fixture struct {
	st                    *store.Store
	owner, property, unit string
	tenant                string
}

func setup(t *testing.T) fixture {
	t.Helper()
	st := store.New(testdb.New(t))
	f := fixture{st: st}
	u, err := st.CreateUser(ctx, "owner@example.com", "Owner", "hash")
	if err != nil {
		t.Fatal(err)
	}
	f.owner = u.ID
	f.property, err = st.CreateProperty(ctx, f.owner, store.PropertyInput{Name: "Elm Court", Kind: "multi_family", AddressLine1: "1 Elm Ct", City: "Salem"})
	if err != nil {
		t.Fatal(err)
	}
	f.unit, err = st.CreateUnit(ctx, f.owner, f.property, store.UnitInput{Label: "1A", Bedrooms: 2, Bathrooms: 1.5, MarketRentCents: 120000})
	if err != nil {
		t.Fatal(err)
	}
	f.tenant, err = st.CreateTenant(ctx, f.owner, store.TenantInput{FirstName: "Ada", LastName: "Lovelace"})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f fixture) lease(t *testing.T, start, end string) string {
	t.Helper()
	id, err := f.st.CreateLease(ctx, f.owner, store.LeaseInput{
		UnitID: f.unit, TenantID: f.tenant, StartDate: d(start), EndDate: d(end),
		RentCents: 120000, DueDay: 1, GraceDays: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestUsersEmailIsCaseInsensitive(t *testing.T) {
	f := setup(t)
	_, err := f.st.CreateUser(ctx, "OWNER@example.com", "Again", "hash")
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	u, err := f.st.UserByEmail(ctx, "Owner@Example.com")
	if err != nil || u.ID != f.owner {
		t.Fatalf("lookup: %v %+v", err, u)
	}
}

func TestOtherOwnersCannotSeeRecords(t *testing.T) {
	f := setup(t)
	other, err := f.st.CreateUser(ctx, "other@example.com", "Other", "hash")
	if err != nil {
		t.Fatal(err)
	}
	asOf := d("2026-01-01")

	if _, err := f.st.GetProperty(ctx, other.ID, f.property, asOf); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("property: %v", err)
	}
	if props, _ := f.st.ListProperties(ctx, other.ID, asOf); len(props) != 0 {
		t.Errorf("listed %d properties", len(props))
	}
	if _, err := f.st.CreateUnit(ctx, other.ID, f.property, store.UnitInput{Label: "X"}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("create unit: %v", err)
	}
	if err := f.st.UpdateTenant(ctx, other.ID, f.tenant, store.TenantInput{FirstName: "X", LastName: "Y"}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("update tenant: %v", err)
	}
	if err := f.st.DeleteProperty(ctx, other.ID, f.property); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("delete property: %v", err)
	}
	_, err = f.st.CreateLease(ctx, other.ID, store.LeaseInput{
		UnitID: f.unit, TenantID: f.tenant, StartDate: asOf, EndDate: asOf.AddMonths(12), RentCents: 1, DueDay: 1,
	})
	if !errors.Is(err, store.ErrConflict) {
		t.Errorf("lease on someone else's unit: %v", err)
	}
}

func TestLeaseOverlapIsRejected(t *testing.T) {
	f := setup(t)
	first := f.lease(t, "2026-01-01", "2026-12-31")

	_, err := f.st.CreateLease(ctx, f.owner, store.LeaseInput{
		UnitID: f.unit, TenantID: f.tenant, StartDate: d("2026-12-31"), EndDate: d("2027-12-30"), RentCents: 1, DueDay: 1,
	})
	var ce *store.ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("expected overlap conflict, got %v", err)
	}

	// Ending the first lease early frees the unit.
	if err := f.st.TerminateLease(ctx, f.owner, first, d("2026-08-31")); err != nil {
		t.Fatal(err)
	}
	f.lease(t, "2026-09-01", "2027-08-31")

	// Updating a lease does not clash with itself.
	l, err := f.st.GetLease(ctx, f.owner, first)
	if err != nil {
		t.Fatal(err)
	}
	err = f.st.UpdateLease(ctx, f.owner, first, store.LeaseInput{
		UnitID: f.unit, TenantID: f.tenant, StartDate: l.StartDate, EndDate: l.EndDate, RentCents: 125000, DueDay: 1,
	})
	if err == nil {
		t.Fatal("extending the old lease's dates over the new one should clash")
	}
}

func TestOccupancyAndUnitCurrentTenant(t *testing.T) {
	f := setup(t)
	f.lease(t, "2026-01-01", "2026-06-30")
	if _, err := f.st.CreateUnit(ctx, f.owner, f.property, store.UnitInput{Label: "1B"}); err != nil {
		t.Fatal(err)
	}

	for asOf, want := range map[string]int{"2025-12-31": 0, "2026-01-01": 1, "2026-06-30": 1, "2026-07-01": 0} {
		o, err := f.st.Occupancy(ctx, f.owner, d(asOf))
		if err != nil {
			t.Fatal(err)
		}
		if o.Units != 2 || o.Occupied != want {
			t.Errorf("%s: %+v, want %d occupied", asOf, o, want)
		}
	}

	units, err := f.st.ListUnits(ctx, f.owner, f.property, d("2026-03-01"))
	if err != nil {
		t.Fatal(err)
	}
	if units[0].CurrentTenant == nil || *units[0].CurrentTenant != "Ada Lovelace" || units[1].CurrentLeaseID != nil {
		t.Errorf("units %+v", units)
	}
	if units[0].Bathrooms != 1.5 {
		t.Errorf("bathrooms %v", units[0].Bathrooms)
	}
}

func TestRentIsInsertedOnce(t *testing.T) {
	f := setup(t)
	id := f.lease(t, "2026-01-01", "2026-12-31")
	l, _ := f.st.GetLease(ctx, f.owner, id)
	rent := domain.RentSchedule(l.Terms(), d("2026-03-15"))

	n, err := f.st.InsertRent(ctx, id, rent)
	if err != nil || n != 3 {
		t.Fatalf("first insert: %d %v", n, err)
	}
	n, err = f.st.InsertRent(ctx, id, rent)
	if err != nil || n != 0 {
		t.Fatalf("second insert: %d %v", n, err)
	}
	charges, _ := f.st.ListCharges(ctx, f.owner, id)
	if len(charges) != 3 || charges[0].PeriodEnd.String() != "2026-01-31" {
		t.Fatalf("charges %+v", charges)
	}

	added, err := f.st.InsertLateFee(ctx, id, charges[0].ID, 5000, d("2026-01-07"))
	if err != nil || !added {
		t.Fatalf("late fee: %v %v", added, err)
	}
	added, _ = f.st.InsertLateFee(ctx, id, charges[0].ID, 5000, d("2026-01-07"))
	if added {
		t.Error("second late fee for the same charge")
	}
	parents, _ := f.st.LateFeeParents(ctx, id)
	if !parents[charges[0].ID] {
		t.Errorf("parents %v", parents)
	}

	// Terminating removes rent due afterwards.
	if err := f.st.TerminateLease(ctx, f.owner, id, d("2026-02-15")); err != nil {
		t.Fatal(err)
	}
	charges, _ = f.st.ListCharges(ctx, f.owner, id)
	for _, c := range charges {
		if c.Kind == domain.ChargeRent && c.DueDate.After(d("2026-02-15")) {
			t.Errorf("rent due %s survived termination", c.DueDate)
		}
	}
}

func TestPaymentsAndLedgerEntries(t *testing.T) {
	f := setup(t)
	id := f.lease(t, "2026-01-01", "2026-12-31")
	for _, on := range []string{"2026-01-02", "2026-02-03", "2026-03-01"} {
		if _, err := f.st.CreatePayment(ctx, f.owner, id, store.PaymentInput{AmountCents: 120000, PaidOn: d(on), Method: "check"}); err != nil {
			t.Fatal(err)
		}
	}
	page, total, err := f.st.ListPayments(ctx, f.owner, store.PaymentFilter{From: d("2026-02-01"), Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(page) != 1 || page[0].PaidOn.String() != "2026-03-01" || page[0].TenantName != "Ada Lovelace" {
		t.Errorf("page %+v total %d", page, total)
	}

	entries, err := f.st.LedgerEntries(ctx, f.owner, "")
	if err != nil {
		t.Fatal(err)
	}
	if e := entries[id]; e == nil || len(e.Payments) != 3 {
		t.Errorf("entries %+v", entries)
	}
}

func TestMaintenanceAndExpenses(t *testing.T) {
	f := setup(t)
	mid, err := f.st.CreateMaintenance(ctx, f.owner, store.MaintenanceInput{
		PropertyID: f.property, UnitID: &f.unit, Title: "Leaky tap", Priority: "high", ReportedOn: d("2026-02-01"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.st.CreateMaintenance(ctx, f.owner, store.MaintenanceInput{
		PropertyID: f.property, Title: "Paint fence", Priority: "low", ReportedOn: d("2026-02-02"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.st.CreateExpense(ctx, f.owner, store.ExpenseInput{
		PropertyID: f.property, MaintenanceRequestID: &mid, Category: "repairs", AmountCents: 8500, SpentOn: d("2026-02-03"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.st.CreateExpense(ctx, f.owner, store.ExpenseInput{
		PropertyID: f.property, Category: "insurance", AmountCents: 50000, SpentOn: d("2026-03-01"),
	}); err != nil {
		t.Fatal(err)
	}

	m, err := f.st.GetMaintenance(ctx, f.owner, mid)
	if err != nil || m.CostCents != 8500 || *m.UnitLabel != "1A" {
		t.Fatalf("request %+v %v", m, err)
	}

	done := d("2026-02-04")
	if err := f.st.SetMaintenanceStatus(ctx, f.owner, mid, domain.MaintenanceCompleted, &done); err != nil {
		t.Fatal(err)
	}
	active, _ := f.st.ListMaintenance(ctx, f.owner, store.MaintenanceFilter{Status: "active"})
	if len(active) != 1 || active[0].Title != "Paint fence" {
		t.Errorf("active %+v", active)
	}
	counts, _ := f.st.MaintenanceCounts(ctx, f.owner)
	if counts["low"] != 1 || counts["high"] != 0 {
		t.Errorf("counts %v", counts)
	}

	totals, _ := f.st.ExpenseTotals(ctx, f.owner, store.ExpenseFilter{})
	if len(totals) != 2 || totals[0].Category != "insurance" {
		t.Errorf("totals %+v", totals)
	}

	other, _ := f.st.CreateProperty(ctx, f.owner, store.PropertyInput{Name: "Other", AddressLine1: "x", City: "y", Kind: "condo"})
	_, err = f.st.CreateExpense(ctx, f.owner, store.ExpenseInput{
		PropertyID: other, MaintenanceRequestID: &mid, Category: "repairs", AmountCents: 1, SpentOn: d("2026-02-03"),
	})
	if !errors.Is(err, store.ErrConflict) {
		t.Errorf("expense linked to another property's request: %v", err)
	}
}

func TestMonthlyTotals(t *testing.T) {
	f := setup(t)
	id := f.lease(t, "2026-01-01", "2026-12-31")
	l, _ := f.st.GetLease(ctx, f.owner, id)
	if _, err := f.st.InsertRent(ctx, id, domain.RentSchedule(l.Terms(), d("2026-02-28"))); err != nil {
		t.Fatal(err)
	}
	if _, err := f.st.CreatePayment(ctx, f.owner, id, store.PaymentInput{AmountCents: 120000, PaidOn: d("2026-02-01"), Method: "cash"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.st.CreateExpense(ctx, f.owner, store.ExpenseInput{PropertyID: f.property, Category: "taxes", AmountCents: 30000, SpentOn: d("2026-02-28")}); err != nil {
		t.Fatal(err)
	}

	months, err := f.st.MonthlyTotals(ctx, f.owner, d("2025-12-01"), d("2026-02-10"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(months) != 3 {
		t.Fatalf("months %+v", months)
	}
	feb := months[2]
	if feb.Month != "2026-02" || feb.ChargedCents != 120000 || feb.CollectedCents != 120000 || feb.ExpensesCents != 30000 {
		t.Errorf("february %+v", feb)
	}
	if months[0].ChargedCents != 0 || months[0].Month != "2025-12" {
		t.Errorf("december %+v", months[0])
	}
}

func TestExpiringLeases(t *testing.T) {
	f := setup(t)
	f.lease(t, "2025-07-01", "2026-03-31")
	got, err := f.st.ExpiringLeases(ctx, f.owner, d("2026-02-15"), 60)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %+v %v", got, err)
	}
	got, _ = f.st.ExpiringLeases(ctx, f.owner, d("2026-01-01"), 60)
	if len(got) != 0 {
		t.Errorf("lease ending in 90 days listed: %+v", got)
	}
}

func TestTenantSearchTreatsWildcardsLiterally(t *testing.T) {
	f := setup(t)
	for _, in := range []store.TenantInput{
		{FirstName: "Ann", LastName: "Lee", Email: "ann_lee@example.com"},
		{FirstName: "Annie", LastName: "Leeds", Email: "annxlee@example.com"},
	} {
		if _, err := f.st.CreateTenant(ctx, f.owner, in); err != nil {
			t.Fatal(err)
		}
	}
	for q, want := range map[string]int{"ann_lee": 1, "%": 0, "ann": 2, "lovelace": 1} {
		got, err := f.st.ListTenants(ctx, f.owner, q)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != want {
			t.Errorf("search %q: %d tenants, want %d", q, len(got), want)
		}
	}
}

func TestChangingTheDueDayReplacesRent(t *testing.T) {
	f := setup(t)
	id := f.lease(t, "2026-01-01", "2026-12-31")
	l, _ := f.st.GetLease(ctx, f.owner, id)
	if _, err := f.st.InsertRent(ctx, id, domain.RentSchedule(l.Terms(), d("2026-03-10"))); err != nil {
		t.Fatal(err)
	}

	err := f.st.UpdateLease(ctx, f.owner, id, store.LeaseInput{
		UnitID: f.unit, TenantID: f.tenant, StartDate: l.StartDate, EndDate: l.EndDate,
		RentCents: l.RentCents, DueDay: 15, GraceDays: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	l, _ = f.st.GetLease(ctx, f.owner, id)
	if _, err := f.st.InsertRent(ctx, id, domain.RentSchedule(l.Terms(), d("2026-03-10"))); err != nil {
		t.Fatal(err)
	}

	// Rent is now due on the 15th: January 1-14 prorated, then January 15
	// and February 15. The old charges on the 1st are gone.
	charges, _ := f.st.ListCharges(ctx, f.owner, id)
	var got []string
	for _, c := range charges {
		got = append(got, c.DueDate.String())
	}
	want := []string{"2026-01-01", "2026-01-15", "2026-02-15"}
	if len(got) != len(want) {
		t.Fatalf("rent due %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rent due %v, want %v", got, want)
		}
	}
	if charges[0].AmountCents == l.RentCents {
		t.Errorf("first period should be prorated, got %d", charges[0].AmountCents)
	}
}

func TestEditingTheRentUpdatesCharges(t *testing.T) {
	f := setup(t)
	id := f.lease(t, "2026-01-01", "2026-12-31")
	l, _ := f.st.GetLease(ctx, f.owner, id)
	if _, err := f.st.InsertRent(ctx, id, domain.RentSchedule(l.Terms(), d("2026-02-10"))); err != nil {
		t.Fatal(err)
	}
	charges, _ := f.st.ListCharges(ctx, f.owner, id)
	if _, err := f.st.InsertLateFee(ctx, id, charges[0].ID, 5000, d("2026-01-07")); err != nil {
		t.Fatal(err)
	}

	// The rent was typed wrong; fixing it fixes the charges and keeps the
	// late fee.
	err := f.st.UpdateLease(ctx, f.owner, id, store.LeaseInput{
		UnitID: f.unit, TenantID: f.tenant, StartDate: l.StartDate, EndDate: l.EndDate,
		RentCents: 102000, DueDay: 1, GraceDays: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	charges, _ = f.st.ListCharges(ctx, f.owner, id)
	if len(charges) != 3 {
		t.Fatalf("charges %+v", charges)
	}
	for _, c := range charges {
		if c.Kind == domain.ChargeRent && c.AmountCents != 102000 {
			t.Errorf("rent due %s is %d", c.DueDate, c.AmountCents)
		}
	}
}

func TestTerminationProratesTheFinalMonth(t *testing.T) {
	f := setup(t)
	id := f.lease(t, "2026-01-01", "2026-12-31")
	l, _ := f.st.GetLease(ctx, f.owner, id)
	if _, err := f.st.InsertRent(ctx, id, domain.RentSchedule(l.Terms(), d("2026-03-10"))); err != nil {
		t.Fatal(err)
	}
	if err := f.st.TerminateLease(ctx, f.owner, id, d("2026-03-15")); err != nil {
		t.Fatal(err)
	}
	charges, _ := f.st.ListCharges(ctx, f.owner, id)
	if len(charges) != 3 {
		t.Fatalf("charges %+v", charges)
	}
	march := charges[2]
	// 15 of March's 31 days of 1,200.00.
	if march.AmountCents != 58065 || march.PeriodEnd.String() != "2026-03-15" {
		t.Errorf("march %d until %s", march.AmountCents, march.PeriodEnd)
	}
	if charges[1].AmountCents != 120000 {
		t.Errorf("february should be untouched, got %d", charges[1].AmountCents)
	}
}

func TestPaymentPageSizeIsCapped(t *testing.T) {
	f := setup(t)
	id := f.lease(t, "2026-01-01", "2026-12-31")
	for i := range 60 {
		if _, err := f.st.CreatePayment(ctx, f.owner, id, store.PaymentInput{
			AmountCents: int64(100 + i), PaidOn: d("2026-01-01").AddDays(i), Method: "cash",
		}); err != nil {
			t.Fatal(err)
		}
	}
	for limit, want := range map[int]int{0: 50, 55: 55, 100: 60, 10000: 60} {
		page, total, err := f.st.ListPayments(ctx, f.owner, store.PaymentFilter{Limit: limit})
		if err != nil {
			t.Fatal(err)
		}
		if len(page) != want || total != 60 {
			t.Errorf("limit %d: %d payments of %d, want %d", limit, len(page), total, want)
		}
	}
}
