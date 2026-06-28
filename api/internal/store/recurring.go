package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/sorin-work/estates/api/internal/domain"
)

type RecurringExpense struct {
	ID               string       `db:"id" json:"id"`
	PropertyID       string       `db:"property_id" json:"property_id"`
	PropertyName     string       `db:"property_name" json:"property_name"`
	VendorID         *string      `db:"vendor_id" json:"vendor_id"`
	Vendor           string       `db:"vendor" json:"vendor"`
	Category         string       `db:"category" json:"category"`
	AmountCents      int64        `db:"amount_cents" json:"amount_cents"`
	Description      string       `db:"description" json:"description"`
	DayOfMonth       int          `db:"day_of_month" json:"day_of_month"`
	EveryMonths      int          `db:"every_months" json:"every_months"`
	StartDate        domain.Date  `db:"start_date" json:"start_date"`
	EndDate          *domain.Date `db:"end_date" json:"end_date"`
	GeneratedThrough *domain.Date `db:"generated_through" json:"generated_through"`
	CreatedAt        time.Time    `db:"created_at" json:"created_at"`
	OwnerID          string       `db:"owner_id" json:"-"`
}

// Recurrence returns when the expense comes back.
func (r RecurringExpense) Recurrence() domain.Recurrence {
	rec := domain.Recurrence{Start: r.StartDate, Day: r.DayOfMonth, EveryMonths: r.EveryMonths}
	if r.EndDate != nil {
		rec.End = *r.EndDate
	}
	return rec
}

type RecurringExpenseInput struct {
	PropertyID  string       `json:"property_id"`
	VendorID    *string      `json:"vendor_id"`
	Vendor      string       `json:"vendor"`
	Category    string       `json:"category"`
	AmountCents int64        `json:"amount_cents"`
	Description string       `json:"description"`
	DayOfMonth  int          `json:"day_of_month"`
	EveryMonths int          `json:"every_months"`
	StartDate   domain.Date  `json:"start_date"`
	EndDate     *domain.Date `json:"end_date"`
}

const recurringSelect = `
	SELECT r.id, r.property_id, p.name AS property_name, r.vendor_id, COALESCE(vd.name, r.vendor) AS vendor,
	       r.category, r.amount_cents, r.description, r.day_of_month, r.every_months, r.start_date,
	       r.end_date, r.generated_through, r.created_at, p.owner_id
	  FROM recurring_expenses r
	  JOIN properties p ON p.id = r.property_id
	  LEFT JOIN vendors vd ON vd.id = r.vendor_id`

func (s *Store) ListRecurringExpenses(ctx context.Context, ownerID, propertyID string) ([]RecurringExpense, error) {
	w := newFilter(ownerID)
	w.raw("p.owner_id = $1")
	if propertyID != "" {
		w.add("r.property_id = ?", propertyID)
	}
	rows, err := s.pool.Query(ctx, recurringSelect+w.where()+` ORDER BY lower(p.name), r.day_of_month, r.category`, w.args...)
	return collectAll[RecurringExpense](rows, err)
}

func (s *Store) GetRecurringExpense(ctx context.Context, ownerID, id string) (RecurringExpense, error) {
	rows, err := s.pool.Query(ctx, recurringSelect+` WHERE p.owner_id = $1 AND r.id = $2`, ownerID, id)
	return collectOne[RecurringExpense](rows, err)
}

func (s *Store) checkRecurringRefs(ctx context.Context, ownerID string, in RecurringExpenseInput) error {
	var ok bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM properties WHERE id = $1 AND owner_id = $2)`,
		in.PropertyID, ownerID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return conflict("property not found")
	}
	return s.checkVendor(ctx, ownerID, in.VendorID)
}

func (s *Store) CreateRecurringExpense(ctx context.Context, ownerID string, in RecurringExpenseInput) (string, error) {
	if err := s.checkRecurringRefs(ctx, ownerID, in); err != nil {
		return "", err
	}
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO recurring_expenses (property_id, vendor_id, vendor, category, amount_cents, description,
		                                day_of_month, every_months, start_date, end_date)
		VALUES ($1, $2, `+vendorNameAt("$3", "$2")+`, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		in.PropertyID, emptyToNil(in.VendorID), in.Vendor, in.Category, in.AmountCents, in.Description,
		in.DayOfMonth, in.EveryMonths, in.StartDate, in.EndDate,
	).Scan(&id)
	return id, err
}

// UpdateRecurringExpense changes the terms. Expenses already created keep
// their amounts; later ones use the new terms.
func (s *Store) UpdateRecurringExpense(ctx context.Context, ownerID, id string, in RecurringExpenseInput) error {
	if err := s.checkRecurringRefs(ctx, ownerID, in); err != nil {
		return err
	}
	return execOne(s.pool.Exec(ctx, `
		UPDATE recurring_expenses r
		   SET property_id = $3, vendor_id = $4, vendor = `+vendorNameAt("$5", "$4")+`, category = $6,
		       amount_cents = $7, description = $8, day_of_month = $9, every_months = $10,
		       start_date = $11, end_date = $12, updated_at = now()
		  FROM properties p
		 WHERE p.id = r.property_id AND p.owner_id = $1 AND r.id = $2`,
		ownerID, id, in.PropertyID, emptyToNil(in.VendorID), in.Vendor, in.Category, in.AmountCents,
		in.Description, in.DayOfMonth, in.EveryMonths, in.StartDate, in.EndDate,
	))
}

// DeleteRecurringExpense stops a recurring expense. Expenses it already
// created are kept.
func (s *Store) DeleteRecurringExpense(ctx context.Context, ownerID, id string) error {
	return execOne(s.pool.Exec(ctx, `
		DELETE FROM recurring_expenses r USING properties p
		 WHERE p.id = r.property_id AND p.owner_id = $1 AND r.id = $2`, ownerID, id))
}

// DueRecurringExpenses returns the recurring expenses of every owner that
// have started by asOf and have not been generated through asOf yet.
func (s *Store) DueRecurringExpenses(ctx context.Context, asOf domain.Date) ([]RecurringExpense, error) {
	rows, err := s.pool.Query(ctx, recurringSelect+`
		WHERE r.start_date <= $1
		  AND (r.generated_through IS NULL OR r.generated_through < LEAST($1, COALESCE(r.end_date, $1)))
		ORDER BY r.id`, asOf)
	return collectAll[RecurringExpense](rows, err)
}

// InsertRecurringOccurrences creates an expense for each date and records
// that the recurring expense has been generated through the last one. It
// returns how many expenses were added.
func (s *Store) InsertRecurringOccurrences(ctx context.Context, r RecurringExpense, dates []domain.Date) (int, error) {
	if len(dates) == 0 {
		return 0, nil
	}
	added := 0
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		for _, d := range dates {
			tag, err := tx.Exec(ctx, `
				INSERT INTO expenses (property_id, vendor_id, vendor, category, amount_cents, spent_on,
				                      description, recurring_expense_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (recurring_expense_id, spent_on) WHERE recurring_expense_id IS NOT NULL DO NOTHING`,
				r.PropertyID, r.VendorID, r.Vendor, r.Category, r.AmountCents, d, r.Description, r.ID)
			if err != nil {
				return err
			}
			added += int(tag.RowsAffected())
		}
		_, err := tx.Exec(ctx, `UPDATE recurring_expenses SET generated_through = $2 WHERE id = $1`, r.ID, dates[len(dates)-1])
		return err
	})
	return added, err
}
