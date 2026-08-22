package httpapi

import (
	"net/http"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type receiptParty struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone,omitempty"`
}

type receiptResponse struct {
	Number   int           `json:"number"`
	IssuedOn domain.Date   `json:"issued_on"`
	Payment  store.Payment `json:"payment"`
	Landlord receiptParty  `json:"landlord"`
	Tenant   receiptParty  `json:"tenant"`
	Property struct {
		Name         string `json:"name"`
		AddressLine1 string `json:"address_line1"`
		AddressLine2 string `json:"address_line2"`
		City         string `json:"city"`
		Region       string `json:"region"`
		PostalCode   string `json:"postal_code"`
		UnitLabel    string `json:"unit_label"`
	} `json:"property"`
	// BalanceAfterCents is what was still owed on the lease once this
	// payment, and any others made the same day, had been counted.
	BalanceAfterCents int64 `json:"balance_after_cents"`
}

// paymentReceipt has everything needed to print a rent receipt.
func (s *Server) paymentReceipt(w http.ResponseWriter, r *http.Request) error {
	ctx, owner := r.Context(), userID(r)
	id, err := pathID(r, "payment")
	if err != nil {
		return err
	}
	p, err := s.store.GetPayment(ctx, owner, id)
	if err != nil {
		return notFoundAs(err, "payment")
	}
	l, err := s.store.GetLease(ctx, owner, p.LeaseID)
	if err != nil {
		return err
	}
	tenant, err := s.store.GetTenant(ctx, owner, l.TenantID)
	if err != nil {
		return err
	}
	prop, err := s.store.GetProperty(ctx, owner, l.PropertyID, s.today())
	if err != nil {
		return err
	}
	landlord, err := s.store.UserByID(ctx, owner)
	if err != nil {
		return err
	}
	entries, err := s.store.LedgerEntries(ctx, owner, l.ID)
	if err != nil {
		return err
	}

	resp := receiptResponse{Number: p.ReceiptNumber, IssuedOn: s.today(), Payment: p}
	resp.Landlord = receiptParty{Name: landlord.Name, Email: landlord.Email}
	resp.Tenant = receiptParty{Name: tenant.FirstName + " " + tenant.LastName, Email: tenant.Email, Phone: tenant.Phone}
	resp.Property.Name = prop.Name
	resp.Property.AddressLine1 = prop.AddressLine1
	resp.Property.AddressLine2 = prop.AddressLine2
	resp.Property.City = prop.City
	resp.Property.Region = prop.Region
	resp.Property.PostalCode = prop.PostalCode
	resp.Property.UnitLabel = l.UnitLabel
	if e := entries[l.ID]; e != nil {
		for _, c := range e.Charges {
			if !c.DueDate.After(p.PaidOn) {
				resp.BalanceAfterCents += c.AmountCents
			}
		}
		for _, pm := range e.Payments {
			if !pm.PaidOn.After(p.PaidOn) {
				resp.BalanceAfterCents -= pm.AmountCents
			}
		}
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}
