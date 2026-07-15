// Package seed creates a demo account with a year of realistic data, so the
// app has something to show the first time it starts.
package seed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/sorin-work/estates/api/internal/auth"
	"github.com/sorin-work/estates/api/internal/billing"
	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

const (
	DemoEmail    = "demo@example.com"
	DemoPassword = "demo-password"
)

// Demo creates the demo user and its data unless the user already exists.
// Dates are relative to today so the dashboard always looks current.
func Demo(ctx context.Context, st *store.Store, bill *billing.Service, log *slog.Logger) error {
	_, err := st.UserByEmail(ctx, DemoEmail)
	if err == nil {
		log.Info("demo user already exists, skipping seed")
		return nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return err
	}

	s := &seeder{st: st, bill: bill, today: domain.Today(), rnd: rand.New(rand.NewPCG(2025, 11))}
	if err := s.run(ctx); err != nil {
		return err
	}
	log.Info("seeded demo data", "email", DemoEmail, "leases", s.leases, "payments", s.payments)
	return nil
}

type seeder struct {
	st    *store.Store
	bill  *billing.Service
	today domain.Date
	rnd   *rand.Rand
	owner string
	// vendors maps vendor names to IDs.
	vendors map[string]string

	leases, payments int
}

// payer describes how a tenant pays, which drives the payment history.
type payer int

const (
	onTime    payer = iota // a day or two either side of the due date
	usuallyOK              // on time, occasionally a week late
	late                   // regularly late and charged a late fee
	behind                 // stopped paying the last two months
	partial                // paid only part of this month's rent
)

type leaseSpec struct {
	unit, tenant string
	// startMonths is how many months before the current month the lease
	// started; lengthMonths is the term.
	startMonths, lengthMonths int
	startDay                  int
	rent, deposit             int64
	dueDay, grace             int
	lateFee                   int64
	payer                     payer
	terminated                bool
	// increase is the rent from the second year of the lease, if it went up.
	increase int64
	// renewal is the rent of an already signed one year renewal.
	renewal int64
}

func (s *seeder) run(ctx context.Context) error {
	hash, err := auth.HashPassword(DemoPassword)
	if err != nil {
		return err
	}
	u, err := s.st.CreateUser(ctx, DemoEmail, "Demo Landlord", hash)
	if err != nil {
		return err
	}
	s.owner = u.ID

	props := map[string]string{}
	units := map[string]string{}
	for _, p := range demoProperties {
		id, err := s.st.CreateProperty(ctx, s.owner, p.in)
		if err != nil {
			return err
		}
		props[p.key] = id
		for _, un := range p.units {
			uid, err := s.st.CreateUnit(ctx, s.owner, id, un.in)
			if err != nil {
				return err
			}
			units[p.key+"/"+un.in.Label] = uid
		}
	}

	tenants := map[string]string{}
	for _, t := range demoTenants {
		id, err := s.st.CreateTenant(ctx, s.owner, t)
		if err != nil {
			return err
		}
		tenants[t.FirstName+" "+t.LastName] = id
	}

	s.vendors = map[string]string{}
	for _, v := range demoVendors {
		id, err := s.st.CreateVendor(ctx, s.owner, v)
		if err != nil {
			return err
		}
		s.vendors[v.Name] = id
	}

	for _, spec := range demoLeases {
		if err := s.lease(ctx, spec, units[spec.unit], tenants[spec.tenant]); err != nil {
			return fmt.Errorf("lease for %s: %w", spec.tenant, err)
		}
	}

	if err := s.maintenance(ctx, props, units); err != nil {
		return err
	}
	return s.expenses(ctx, props)
}

// vendorID returns the ID of a demo vendor, or nil for names that are not
// vendors.
func (s *seeder) vendorID(name string) *string {
	id, ok := s.vendors[name]
	if !ok {
		return nil
	}
	return &id
}

func (s *seeder) monthsAgo(n, day int) domain.Date {
	m := s.today.MonthStart().AddMonths(-n)
	return domain.NewDate(m.Year(), m.Month(), min(day, domain.DaysIn(m.Year(), m.Month())))
}

func (s *seeder) lease(ctx context.Context, spec leaseSpec, unitID, tenantID string) error {
	start := s.monthsAgo(spec.startMonths, spec.startDay)
	end := start.AddMonths(spec.lengthMonths).AddDays(-1)
	in := store.LeaseInput{
		UnitID: unitID, TenantID: tenantID,
		StartDate: start, EndDate: end,
		RentCents: spec.rent, DepositCents: spec.deposit,
		DueDay: spec.dueDay, GraceDays: spec.grace, LateFeeCents: spec.lateFee,
	}
	id, err := s.st.CreateLease(ctx, s.owner, in)
	if err != nil {
		return err
	}
	s.leases++

	if spec.terminated {
		if err := s.st.TerminateLease(ctx, s.owner, id, s.today.AddDays(-75)); err != nil {
			return err
		}
	}
	if spec.deposit > 0 {
		if err := s.deposit(ctx, id, spec, start); err != nil {
			return err
		}
	}
	if spec.increase > 0 {
		if _, err := s.st.AddRentChange(ctx, s.owner, id, store.RentChangeInput{
			EffectiveDate: start.AddMonths(12), RentCents: spec.increase, Notes: "Annual increase",
		}); err != nil {
			return err
		}
	}

	l, err := s.st.GetLease(ctx, s.owner, id)
	if err != nil {
		return err
	}
	// Generate rent without late fees; the payment history decides which
	// late fees are charged.
	rent := domain.RentSchedule(l.Terms(), s.today)
	if _, err := s.st.InsertRent(ctx, id, rent); err != nil {
		return err
	}

	for i, r := range rent {
		monthsFromEnd := len(rent) - 1 - i
		paidOn := r.DueDate
		amount := r.AmountCents
		switch spec.payer {
		case onTime:
			paidOn = paidOn.AddDays(s.rnd.IntN(4) - 2)
		case usuallyOK:
			paidOn = paidOn.AddDays(s.rnd.IntN(3))
			if s.rnd.IntN(5) == 0 {
				paidOn = paidOn.AddDays(4 + s.rnd.IntN(4))
			}
		case late:
			paidOn = paidOn.AddDays(spec.grace + 2 + s.rnd.IntN(8))
		case behind:
			if monthsFromEnd < 2 {
				continue
			}
			paidOn = paidOn.AddDays(s.rnd.IntN(6))
		case partial:
			paidOn = paidOn.AddDays(s.rnd.IntN(3))
			if monthsFromEnd == 0 {
				amount = amount * 3 / 5
			}
		}
		if paidOn.After(s.today) {
			// Not paid yet; it is still within the grace period.
			continue
		}
		if spec.payer == late && spec.lateFee > 0 {
			feeDue := r.DueDate.AddDays(spec.grace + 1)
			if _, err := s.st.CreateCharge(ctx, s.owner, id, store.ChargeInput{
				Kind: domain.ChargeLateFee, Description: "Late fee for rent due " + r.DueDate.String(),
				AmountCents: spec.lateFee, DueDate: feeDue,
			}); err != nil {
				return err
			}
			amount += spec.lateFee
		}
		if err := s.pay(ctx, id, amount, paidOn, ""); err != nil {
			return err
		}
	}

	// Late fees for anything still overdue today.
	if _, err := s.bill.BillLease(ctx, l, s.today); err != nil {
		return err
	}
	if spec.renewal > 0 {
		_, err = s.st.RenewLease(ctx, s.owner, id, store.RenewalInput{
			EndDate: end.AddMonths(12), RentCents: spec.renewal, Notes: "Renewed for another year.",
		})
	}
	return err
}

// deposit records the deposit paid before move-in. Marcus Reed moved out
// early: part of his deposit paid for cleaning and the rest went back.
func (s *seeder) deposit(ctx context.Context, leaseID string, spec leaseSpec, start domain.Date) error {
	received := start.AddDays(-3)
	if received.After(s.today) {
		return nil
	}
	if _, err := s.st.CreateDeposit(ctx, s.owner, leaseID, store.DepositInput{
		Kind: domain.DepositReceived, AmountCents: spec.deposit, OccurredOn: received, Notes: "Paid at signing",
	}); err != nil {
		return err
	}
	if !spec.terminated {
		return nil
	}
	settled := s.today.AddDays(-60)
	const cleaning = 32500
	if _, err := s.st.CreateDeposit(ctx, s.owner, leaseID, store.DepositInput{
		Kind: domain.DepositWithheld, AmountCents: cleaning, OccurredOn: settled, Notes: "Carpet cleaning and wall repair",
	}); err != nil {
		return err
	}
	_, err := s.st.CreateDeposit(ctx, s.owner, leaseID, store.DepositInput{
		Kind: domain.DepositReturned, AmountCents: spec.deposit - cleaning, OccurredOn: settled, Notes: "Refund by check",
	})
	return err
}

var methods = []string{"bank_transfer", "bank_transfer", "bank_transfer", "check", "card"}

func (s *seeder) pay(ctx context.Context, leaseID string, amount int64, on domain.Date, notes string) error {
	if on.After(s.today) {
		on = s.today
	}
	method := methods[s.rnd.IntN(len(methods))]
	ref := ""
	if method == "check" {
		ref = fmt.Sprintf("Check #%d", 1000+s.rnd.IntN(9000))
	}
	_, err := s.st.CreatePayment(ctx, s.owner, leaseID, store.PaymentInput{
		AmountCents: amount, PaidOn: on, Method: method, Reference: ref, Notes: notes,
	})
	if err == nil {
		s.payments++
	}
	return err
}

func (s *seeder) maintenance(ctx context.Context, props, units map[string]string) error {
	for _, m := range demoMaintenance {
		var unitID *string
		if m.unit != "" {
			id := units[m.prop+"/"+m.unit]
			unitID = &id
		}
		reported := s.today.AddDays(-m.daysAgo)
		id, err := s.st.CreateMaintenance(ctx, s.owner, store.MaintenanceInput{
			PropertyID: props[m.prop], UnitID: unitID, Title: m.title, Description: m.description,
			Priority: m.priority, Vendor: m.vendor, VendorID: s.vendorID(m.vendor), ReportedOn: reported,
		})
		if err != nil {
			return err
		}
		if m.status != domain.MaintenanceOpen {
			var done *domain.Date
			if m.status == domain.MaintenanceCompleted {
				d := reported.AddDays(m.daysToFix)
				done = &d
			}
			if err := s.st.SetMaintenanceStatus(ctx, s.owner, id, m.status, done); err != nil {
				return err
			}
		}
		if m.cost > 0 {
			spent := reported.AddDays(m.daysToFix)
			if spent.After(s.today) {
				spent = s.today
			}
			if _, err := s.st.CreateExpense(ctx, s.owner, store.ExpenseInput{
				PropertyID: props[m.prop], MaintenanceRequestID: &id, Category: "repairs",
				AmountCents: m.cost, SpentOn: spent, Vendor: m.vendor, VendorID: s.vendorID(m.vendor), Description: m.title,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *seeder) expenses(ctx context.Context, props map[string]string) error {
	add := func(prop, category string, amount int64, on domain.Date, vendor, desc string) error {
		if on.After(s.today) {
			return nil
		}
		_, err := s.st.CreateExpense(ctx, s.owner, store.ExpenseInput{
			PropertyID: props[prop], Category: category, AmountCents: amount, SpentOn: on,
			Vendor: vendor, VendorID: s.vendorID(vendor), Description: desc,
		})
		return err
	}
	jitter := func(base int64, pct int) int64 {
		return base + base*int64(s.rnd.IntN(2*pct+1)-pct)/100
	}

	// Fixed monthly costs are recurring expenses, created by the billing
	// service for the last twelve months.
	recurring := []store.RecurringExpenseInput{
		{PropertyID: props["harbor"], Vendor: "First Coastal Bank", Category: "mortgage", AmountCents: 685000, Description: "Mortgage payment", DayOfMonth: 1},
		{PropertyID: props["birch"], Vendor: "First Coastal Bank", Category: "mortgage", AmountCents: 142000, Description: "Mortgage payment", DayOfMonth: 1},
		{PropertyID: props["harbor"], Vendor: "GreenLeaf Landscaping", Category: "maintenance", AmountCents: 18000, Description: "Grounds and snow service", DayOfMonth: 20},
		{PropertyID: props["riverside"], Vendor: "Riverside Condo Association", Category: "hoa", AmountCents: 38500, Description: "HOA dues", DayOfMonth: 5},
	}
	for _, in := range recurring {
		in.VendorID = s.vendorID(in.Vendor)
		in.EveryMonths = 1
		in.StartDate = s.monthsAgo(11, 1)
		id, err := s.st.CreateRecurringExpense(ctx, s.owner, in)
		if err != nil {
			return err
		}
		re, err := s.st.GetRecurringExpense(ctx, s.owner, id)
		if err != nil {
			return err
		}
		if _, err := s.bill.GenerateRecurring(ctx, re, s.today); err != nil {
			return err
		}
	}

	for i := 11; i >= 0; i-- {
		steps := []error{
			add("harbor", "utilities", jitter(41000, 15), s.monthsAgo(i, 12), "City Water & Sewer", "Water and sewer, common meter"),
			add("maple", "utilities", jitter(16500, 20), s.monthsAgo(i, 12), "City Water & Sewer", "Water and sewer"),
		}
		if err := errors.Join(steps...); err != nil {
			return err
		}
	}

	yearly := []error{
		add("harbor", "insurance", 384000, s.monthsAgo(9, 15), "Northwest Mutual Property", "Annual building insurance"),
		add("maple", "insurance", 162000, s.monthsAgo(7, 3), "Northwest Mutual Property", "Annual landlord policy"),
		add("birch", "insurance", 118000, s.monthsAgo(4, 22), "Northwest Mutual Property", "Annual landlord policy"),
		add("riverside", "insurance", 46000, s.monthsAgo(2, 10), "Northwest Mutual Property", "HO-6 policy"),
		add("harbor", "taxes", 512000, s.monthsAgo(10, 15), "Multnomah County", "Property tax, first half"),
		add("harbor", "taxes", 512000, s.monthsAgo(4, 15), "Multnomah County", "Property tax, second half"),
		add("maple", "taxes", 211000, s.monthsAgo(10, 15), "Multnomah County", "Property tax, first half"),
		add("maple", "taxes", 211000, s.monthsAgo(4, 15), "Multnomah County", "Property tax, second half"),
		add("birch", "taxes", 289000, s.monthsAgo(5, 28), "Washington County", "Annual property tax"),
		add("riverside", "taxes", 174000, s.monthsAgo(5, 28), "Multnomah County", "Annual property tax"),
		add("harbor", "legal", 45000, s.monthsAgo(6, 9), "Hale & Ortiz LLP", "Lease template review"),
		add("maple", "other", 8900, s.monthsAgo(3, 17), "Hardware Depot", "Smoke detector batteries and filters"),
	}
	return errors.Join(yearly...)
}
