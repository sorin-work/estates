package seed

import "github.com/sorin-work/estates/api/internal/store"

type unitSpec struct{ in store.UnitInput }

type propertySpec struct {
	key   string
	in    store.PropertyInput
	units []unitSpec
}

func sqft(n int) *int { return &n }

func unit(label string, beds int, baths float64, area int, rent int64) unitSpec {
	return unitSpec{store.UnitInput{Label: label, Bedrooms: beds, Bathrooms: baths, SquareFeet: sqft(area), MarketRentCents: rent}}
}

var demoProperties = []propertySpec{
	{
		key: "maple",
		in: store.PropertyInput{
			Name: "Maple Street Duplex", Kind: "multi_family",
			AddressLine1: "2218 SE Maple St", City: "Portland", Region: "OR", PostalCode: "97214",
			Notes: "Bought 2019. Shared laundry in the basement, separate gas meters.",
		},
		units: []unitSpec{unit("A", 2, 1, 950, 185000), unit("B", 2, 1, 950, 185000)},
	},
	{
		key: "harbor",
		in: store.PropertyInput{
			Name: "Harbor View Apartments", Kind: "multi_family",
			AddressLine1: "410 N Harbor Way", City: "Portland", Region: "OR", PostalCode: "97217",
			Notes: "Six units over two floors. Boiler serviced every October.",
		},
		units: []unitSpec{
			unit("101", 1, 1, 640, 139500), unit("102", 1, 1, 640, 139500), unit("103", 2, 1, 820, 165000),
			unit("201", 1, 1, 640, 145000), unit("202", 2, 1.5, 860, 172500), unit("203", 2, 1.5, 860, 172500),
		},
	},
	{
		key: "birch",
		in: store.PropertyInput{
			Name: "14 Birch Lane", Kind: "single_family",
			AddressLine1: "14 Birch Ln", City: "Beaverton", Region: "OR", PostalCode: "97005",
			Notes: "Tenant handles yard work. Garage is excluded from the lease.",
		},
		units: []unitSpec{unit("House", 3, 2, 1650, 265000)},
	},
	{
		key: "riverside",
		in: store.PropertyInput{
			Name: "Riverside Condo 4B", Kind: "condo",
			AddressLine1: "1200 NW Riverside Dr", AddressLine2: "Unit 4B", City: "Portland", Region: "OR", PostalCode: "97209",
			Notes: "HOA covers water, trash and building insurance.",
		},
		units: []unitSpec{unit("4B", 1, 1, 710, 158000)},
	},
}

var demoTenants = []store.TenantInput{
	{FirstName: "Maria", LastName: "Gonzalez", Email: "maria.gonzalez@example.com", Phone: "(503) 555-0142"},
	{FirstName: "James", LastName: "Whitaker", Email: "jwhitaker@example.com", Phone: "(503) 555-0187"},
	{FirstName: "Priya", LastName: "Raman", Email: "priya.raman@example.com", Phone: "(971) 555-0119"},
	{FirstName: "Tom", LastName: "Becker", Email: "tom.becker@example.com", Phone: "(503) 555-0163", Notes: "Prefers text messages."},
	{FirstName: "Aisha", LastName: "Okafor", Email: "aisha.okafor@example.com", Phone: "(503) 555-0108"},
	{FirstName: "Daniel", LastName: "Kim", Email: "dkim@example.com", Phone: "(971) 555-0134"},
	{FirstName: "Sofia", LastName: "Rossi", Email: "sofia.rossi@example.com", Phone: "(503) 555-0175", Notes: "Has a small dog (approved, pet deposit paid)."},
	{FirstName: "Kevin", LastName: "O'Brien", Email: "kevin.obrien@example.com", Phone: "(503) 555-0191"},
	{FirstName: "Hannah", LastName: "Larsen", Email: "hannah.larsen@example.com", Phone: "(971) 555-0156"},
	{FirstName: "Marcus", LastName: "Reed", Email: "marcus.reed@example.com", Phone: "(503) 555-0127", Notes: "Moved out early for a job in Seattle."},
	{FirstName: "Elena", LastName: "Petrova", Email: "elena.petrova@example.com", Phone: "(503) 555-0149", Notes: "Signed for next month, moving from Eugene."},
}

var demoLeases = []leaseSpec{
	{unit: "maple/A", tenant: "Maria Gonzalez", startMonths: 20, lengthMonths: 24, startDay: 1, rent: 180000, deposit: 180000, dueDay: 1, grace: 5, lateFee: 7500, payer: onTime},
	{unit: "maple/B", tenant: "James Whitaker", startMonths: 10, lengthMonths: 12, startDay: 15, rent: 182500, deposit: 182500, dueDay: 1, grace: 5, lateFee: 7500, payer: late},
	{unit: "harbor/101", tenant: "Priya Raman", startMonths: 11, lengthMonths: 12, startDay: 1, rent: 135000, deposit: 135000, dueDay: 1, grace: 3, lateFee: 5000, payer: onTime},
	{unit: "harbor/102", tenant: "Tom Becker", startMonths: 8, lengthMonths: 12, startDay: 1, rent: 139500, deposit: 100000, dueDay: 1, grace: 3, lateFee: 5000, payer: behind},
	{unit: "harbor/103", tenant: "Aisha Okafor", startMonths: 14, lengthMonths: 16, startDay: 1, rent: 162500, deposit: 162500, dueDay: 1, grace: 3, lateFee: 5000, payer: usuallyOK},
	{unit: "harbor/201", tenant: "Daniel Kim", startMonths: 5, lengthMonths: 12, startDay: 10, rent: 145000, deposit: 145000, dueDay: 1, grace: 3, lateFee: 5000, payer: onTime},
	{unit: "harbor/202", tenant: "Sofia Rossi", startMonths: 7, lengthMonths: 12, startDay: 1, rent: 172500, deposit: 197500, dueDay: 1, grace: 3, lateFee: 5000, payer: partial},
	{unit: "harbor/203", tenant: "Marcus Reed", startMonths: 12, lengthMonths: 12, startDay: 1, rent: 168000, deposit: 168000, dueDay: 1, grace: 3, lateFee: 5000, payer: onTime, terminated: true},
	{unit: "birch/House", tenant: "Kevin O'Brien", startMonths: 23, lengthMonths: 24, startDay: 1, rent: 255000, deposit: 300000, dueDay: 5, grace: 5, lateFee: 10000, payer: onTime},
	{unit: "riverside/4B", tenant: "Hannah Larsen", startMonths: 9, lengthMonths: 12, startDay: 1, rent: 155000, deposit: 155000, dueDay: 1, grace: 5, lateFee: 5000, payer: usuallyOK},
	{unit: "harbor/203", tenant: "Elena Petrova", startMonths: -1, lengthMonths: 12, startDay: 1, rent: 172500, deposit: 172500, dueDay: 1, grace: 3, lateFee: 5000, payer: onTime},
}

// demoVendors are the trades people the demo landlord calls. Utilities,
// banks and the county are left as plain names on their expenses.
var demoVendors = []store.VendorInput{
	{Name: "Rapid Rooter Plumbing", Trade: "plumbing", ContactName: "Luis Ortega", Phone: "(503) 555-0110", Email: "dispatch@rapidrooter.example"},
	{Name: "Comfort Air HVAC", Trade: "hvac", ContactName: "Dana Wells", Phone: "(503) 555-0122", Notes: "Annual service contract for the Harbor boiler."},
	{Name: "Fresh Coat Painting", Trade: "painting", ContactName: "Sam Price", Phone: "(971) 555-0133"},
	{Name: "GreenLeaf Landscaping", Trade: "landscaping", Phone: "(503) 555-0144", Email: "office@greenleaf.example", Notes: "Monthly grounds and snow service at Harbor View."},
	{Name: "Appliance Pros", Trade: "appliances", Phone: "(503) 555-0155"},
	{Name: "Handy Hank Repairs", Trade: "handyman", ContactName: "Hank Morrow", Phone: "(971) 555-0166", Notes: "Good for small jobs, books up a week ahead."},
}

type maintenanceSpec struct {
	prop, unit         string
	title, description string
	priority, status   string
	vendor             string
	daysAgo, daysToFix int
	cost               int64
}

var demoMaintenance = []maintenanceSpec{
	{prop: "harbor", unit: "202", title: "Water leaking under kitchen sink", description: "Tenant reports a steady drip and a soft cabinet floor. Water shut off at the valve.", priority: "urgent", status: "in_progress", vendor: "Rapid Rooter Plumbing", daysAgo: 2},
	{prop: "maple", unit: "B", title: "Bedroom window won't latch", description: "Latch is bent; window can't be locked.", priority: "high", status: "open", daysAgo: 4},
	{prop: "harbor", title: "Hallway light out on 2nd floor", priority: "normal", status: "open", daysAgo: 6},
	{prop: "birch", unit: "House", title: "Annual furnace service", description: "Book before the first cold snap.", priority: "normal", status: "on_hold", vendor: "Comfort Air HVAC", daysAgo: 12},
	{prop: "riverside", unit: "4B", title: "Dishwasher not draining", priority: "normal", status: "open", daysAgo: 1},
	{prop: "harbor", unit: "101", title: "Replace bathroom exhaust fan", description: "Fan is very loud and barely pulls air.", priority: "low", status: "open", daysAgo: 20},
	{prop: "harbor", title: "Boiler annual inspection", priority: "high", status: "completed", vendor: "Comfort Air HVAC", daysAgo: 40, daysToFix: 3, cost: 42500},
	{prop: "maple", unit: "A", title: "Garbage disposal jammed", priority: "normal", status: "completed", vendor: "Rapid Rooter Plumbing", daysAgo: 75, daysToFix: 2, cost: 18500},
	{prop: "harbor", unit: "203", title: "Turnover: paint and carpet clean", description: "Between Marcus Reed moving out and the next tenant.", priority: "normal", status: "completed", vendor: "Fresh Coat Painting", daysAgo: 70, daysToFix: 9, cost: 186000},
	{prop: "birch", unit: "House", title: "Gutter cleaning", priority: "low", status: "completed", vendor: "GreenLeaf Landscaping", daysAgo: 160, daysToFix: 6, cost: 22500},
	{prop: "harbor", unit: "103", title: "Fridge not cooling", description: "Compressor replaced under warranty, labor only.", priority: "high", status: "completed", vendor: "Appliance Pros", daysAgo: 210, daysToFix: 1, cost: 14500},
	{prop: "maple", title: "Front steps handrail loose", priority: "high", status: "completed", vendor: "Handy Hank Repairs", daysAgo: 290, daysToFix: 4, cost: 31000},
	{prop: "riverside", unit: "4B", title: "Repaint accent wall", description: "Tenant asked to paint the living room; decided to leave as is.", priority: "low", status: "cancelled", daysAgo: 120},
}
