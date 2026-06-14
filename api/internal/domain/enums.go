package domain

// Property kinds.
var PropertyKinds = []string{"single_family", "multi_family", "condo", "townhouse", "commercial", "other"}

// Payment methods.
var PaymentMethods = []string{"bank_transfer", "check", "cash", "card", "other"}

// Expense categories.
var ExpenseCategories = []string{
	"repairs", "maintenance", "utilities", "insurance", "taxes", "mortgage", "management", "hoa", "legal", "other",
}

// Currencies are the currencies amounts can be shown in. All of them have
// two decimal places, so cents work the same way for each.
var Currencies = []string{"USD", "CAD", "EUR", "GBP", "AUD", "NZD", "CHF", "SEK", "PLN", "RON"}

// OneOf reports whether v is in options.
func OneOf(v string, options []string) bool {
	for _, o := range options {
		if o == v {
			return true
		}
	}
	return false
}

// Vendor trades.
var VendorTrades = []string{
	"plumbing", "electrical", "hvac", "appliances", "carpentry", "painting", "cleaning",
	"landscaping", "roofing", "pest_control", "locksmith", "handyman", "other",
}
