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

type receiptPlace struct {
	Name         string `json:"name"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	Region       string `json:"region"`
	PostalCode   string `json:"postal_code"`
	UnitLabel    string `json:"unit_label"`
}

// letterhead is who and where a printed receipt or statement is about.
type letterhead struct {
	Landlord receiptParty `json:"landlord"`
	Tenant   receiptParty `json:"tenant"`
	Property receiptPlace `json:"property"`
}

// loadLetterhead reads the landlord, the tenant and the property of a lease.
func (s *Server) loadLetterhead(r *http.Request, l store.Lease) (letterhead, error) {
	ctx, owner := r.Context(), userID(r)
	var h letterhead
	tenant, err := s.store.GetTenant(ctx, owner, l.TenantID)
	if err != nil {
		return h, err
	}
	prop, err := s.store.GetProperty(ctx, owner, l.PropertyID, s.today(r))
	if err != nil {
		return h, err
	}
	landlord, err := s.store.UserByID(ctx, owner)
	if err != nil {
		return h, err
	}
	h.Landlord = receiptParty{Name: landlord.Name, Email: landlord.Email}
	h.Tenant = receiptParty{Name: tenant.FirstName + " " + tenant.LastName, Email: tenant.Email, Phone: tenant.Phone}
	h.Property = receiptPlace{
		Name: prop.Name, AddressLine1: prop.AddressLine1, AddressLine2: prop.AddressLine2,
		City: prop.City, Region: prop.Region, PostalCode: prop.PostalCode, UnitLabel: l.UnitLabel,
	}
	return h, nil
}

type receiptResponse struct {
	Number   int           `json:"number"`
	IssuedOn domain.Date   `json:"issued_on"`
	Payment  store.Payment `json:"payment"`
	letterhead
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
	head, err := s.loadLetterhead(r, l)
	if err != nil {
		return err
	}
	entries, err := s.store.LedgerEntries(ctx, owner, l.ID)
	if err != nil {
		return err
	}

	resp := receiptResponse{Number: p.ReceiptNumber, IssuedOn: s.today(r), Payment: p, letterhead: head}
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
