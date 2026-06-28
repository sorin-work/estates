package store

import (
	"context"
	"time"

	"github.com/sorin-work/estates/api/internal/domain"
)

type Expense struct {
	ID                   string      `db:"id" json:"id"`
	PropertyID           string      `db:"property_id" json:"property_id"`
	PropertyName         string      `db:"property_name" json:"property_name"`
	MaintenanceRequestID *string     `db:"maintenance_request_id" json:"maintenance_request_id"`
	MaintenanceTitle     *string     `db:"maintenance_title" json:"maintenance_title"`
	Category             string      `db:"category" json:"category"`
	AmountCents          int64       `db:"amount_cents" json:"amount_cents"`
	SpentOn              domain.Date `db:"spent_on" json:"spent_on"`
	Vendor               string      `db:"vendor" json:"vendor"`
	VendorID             *string     `db:"vendor_id" json:"vendor_id"`
	Description          string      `db:"description" json:"description"`
	RecurringExpenseID   *string     `db:"recurring_expense_id" json:"recurring_expense_id"`
	CreatedAt            time.Time   `db:"created_at" json:"created_at"`
}

type ExpenseInput struct {
	PropertyID           string      `json:"property_id"`
	MaintenanceRequestID *string     `json:"maintenance_request_id"`
	Category             string      `json:"category"`
	AmountCents          int64       `json:"amount_cents"`
	SpentOn              domain.Date `json:"spent_on"`
	Vendor               string      `json:"vendor"`
	VendorID             *string     `json:"vendor_id"`
	Description          string      `json:"description"`
}

type ExpenseFilter struct {
	PropertyID string
	Category   string
	VendorID   string
	From       domain.Date
	To         domain.Date
}

// CategoryTotal is the amount spent in one expense category.
type CategoryTotal struct {
	Category    string `db:"category" json:"category"`
	AmountCents int64  `db:"amount_cents" json:"amount_cents"`
}

const expenseSelect = `
	SELECT e.id, e.property_id, p.name AS property_name, e.maintenance_request_id,
	       m.title AS maintenance_title, e.category, e.amount_cents, e.spent_on,
	       COALESCE(vd.name, e.vendor) AS vendor, e.vendor_id, e.description, e.recurring_expense_id,
	       e.created_at
	  FROM expenses e
	  JOIN properties p ON p.id = e.property_id AND p.owner_id = $1
	  LEFT JOIN maintenance_requests m ON m.id = e.maintenance_request_id
	  LEFT JOIN vendors vd ON vd.id = e.vendor_id`

func expenseFilter(ownerID string, f ExpenseFilter) *filter {
	w := newFilter(ownerID)
	if f.PropertyID != "" {
		w.add("e.property_id = ?", f.PropertyID)
	}
	if f.Category != "" {
		w.add("e.category = ?", f.Category)
	}
	if f.VendorID != "" {
		w.add("e.vendor_id = ?", f.VendorID)
	}
	if !f.From.IsZero() {
		w.add("e.spent_on >= ?", f.From)
	}
	if !f.To.IsZero() {
		w.add("e.spent_on <= ?", f.To)
	}
	return w
}

func (s *Store) ListExpenses(ctx context.Context, ownerID string, f ExpenseFilter) ([]Expense, error) {
	w := expenseFilter(ownerID, f)
	rows, err := s.pool.Query(ctx, expenseSelect+w.where()+` ORDER BY e.spent_on DESC, e.created_at DESC`, w.args...)
	return collectAll[Expense](rows, err)
}

// ExpenseTotals sums the filtered expenses by category, largest first.
func (s *Store) ExpenseTotals(ctx context.Context, ownerID string, f ExpenseFilter) ([]CategoryTotal, error) {
	w := expenseFilter(ownerID, f)
	rows, err := s.pool.Query(ctx, `
		SELECT e.category, sum(e.amount_cents)::bigint AS amount_cents
		  FROM expenses e
		  JOIN properties p ON p.id = e.property_id AND p.owner_id = $1`+w.where()+`
		 GROUP BY e.category
		 ORDER BY amount_cents DESC`, w.args...)
	return collectAll[CategoryTotal](rows, err)
}

func (s *Store) GetExpense(ctx context.Context, ownerID, id string) (Expense, error) {
	rows, err := s.pool.Query(ctx, expenseSelect+` WHERE e.id = $2`, ownerID, id)
	return collectOne[Expense](rows, err)
}

// checkExpenseRefs makes sure the property is the owner's and that a linked
// maintenance request is on the same property.
func (s *Store) checkExpenseRefs(ctx context.Context, ownerID string, in ExpenseInput) error {
	var propertyOK, requestOK bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM properties WHERE id = $2 AND owner_id = $1),
		       $3::uuid IS NULL OR EXISTS (SELECT 1 FROM maintenance_requests WHERE id = $3 AND property_id = $2)`,
		ownerID, in.PropertyID, emptyToNil(in.MaintenanceRequestID),
	).Scan(&propertyOK, &requestOK)
	if err != nil {
		return err
	}
	if !propertyOK {
		return conflict("property not found")
	}
	if !requestOK {
		return conflict("the maintenance request is not on this property")
	}
	return s.checkVendor(ctx, ownerID, in.VendorID)
}

func (s *Store) CreateExpense(ctx context.Context, ownerID string, in ExpenseInput) (string, error) {
	if err := s.checkExpenseRefs(ctx, ownerID, in); err != nil {
		return "", err
	}
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO expenses (property_id, maintenance_request_id, category, amount_cents, spent_on, vendor, vendor_id, description)
		VALUES ($1, $2, $3, $4, $5, `+vendorNameAt("$6", "$7")+`, $7, $8)
		RETURNING id`,
		in.PropertyID, emptyToNil(in.MaintenanceRequestID), in.Category, in.AmountCents, in.SpentOn, in.Vendor,
		emptyToNil(in.VendorID), in.Description,
	).Scan(&id)
	return id, err
}

func (s *Store) UpdateExpense(ctx context.Context, ownerID, id string, in ExpenseInput) error {
	if err := s.checkExpenseRefs(ctx, ownerID, in); err != nil {
		return err
	}
	return execOne(s.pool.Exec(ctx, `
		UPDATE expenses e
		   SET property_id = $3, maintenance_request_id = $4, category = $5, amount_cents = $6,
		       spent_on = $7, vendor = `+vendorNameAt("$8", "$9")+`, vendor_id = $9, description = $10,
		       updated_at = now()
		  FROM properties p
		 WHERE p.id = e.property_id AND p.owner_id = $1 AND e.id = $2`,
		ownerID, id, in.PropertyID, emptyToNil(in.MaintenanceRequestID), in.Category, in.AmountCents,
		in.SpentOn, in.Vendor, emptyToNil(in.VendorID), in.Description,
	))
}

func (s *Store) DeleteExpense(ctx context.Context, ownerID, id string) error {
	return execOne(s.pool.Exec(ctx, `
		DELETE FROM expenses e USING properties p
		 WHERE p.id = e.property_id AND p.owner_id = $1 AND e.id = $2`, ownerID, id))
}
