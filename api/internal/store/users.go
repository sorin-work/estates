package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID           string `db:"id" json:"id"`
	Email        string `db:"email" json:"email"`
	Name         string `db:"name" json:"name"`
	PasswordHash string `db:"password_hash" json:"-"`
	Currency     string `db:"currency" json:"currency"`
	// DepositReturnDays is the deadline for returning a deposit after the
	// tenant moves out.
	DepositReturnDays int `db:"deposit_return_days" json:"deposit_return_days"`
	// TokenVersion goes up when the password changes; older tokens stop
	// working.
	TokenVersion int `db:"token_version" json:"-"`
	// TimeZone is an IANA name such as "Europe/Bucharest". It decides what
	// "today" is for this landlord.
	TimeZone  string    `db:"time_zone" json:"time_zone"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

const userColumns = `id, email, name, password_hash, currency, deposit_return_days, token_version, time_zone, created_at`

// CreateUser inserts a user. It returns ErrConflict if the email is taken.
func (s *Store) CreateUser(ctx context.Context, email, name, passwordHash string) (User, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO users (email, name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING `+userColumns, email, name, passwordHash)
	u, err := collectOne[User](rows, err)
	if isUniqueViolation(err) {
		return User{}, conflict("an account with this email already exists")
	}
	return u, err
}

// ErrNotFirstUser means CreateFirstUser found an account already there.
var ErrNotFirstUser = errors.New("an account already exists")

// CreateFirstUser inserts a user only if there is none yet. The table lock
// makes two concurrent calls on an empty database create one account, not
// two.
func (s *Store) CreateFirstUser(ctx context.Context, email, name, passwordHash string) (User, error) {
	var u User
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users)`).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrNotFirstUser
		}
		rows, err := tx.Query(ctx, `
			INSERT INTO users (email, name, password_hash)
			VALUES ($1, $2, $3)
			RETURNING `+userColumns, email, name, passwordHash)
		u, err = collectOne[User](rows, err)
		return err
	})
	if isUniqueViolation(err) {
		return User{}, conflict("an account with this email already exists")
	}
	return u, err
}

// TimeZones returns every time zone at least one account uses.
func (s *Store) TimeZones(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT time_zone FROM users ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// HasUsers reports whether any account exists.
func (s *Store) HasUsers(ctx context.Context) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users)`).Scan(&ok)
	return ok, err
}

// UserByEmail looks a user up by email, ignoring case.
func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+userColumns+` FROM users WHERE lower(email) = lower($1)`, email)
	return collectOne[User](rows, err)
}

func (s *Store) UserByID(ctx context.Context, id string) (User, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return collectOne[User](rows, err)
}

// UserSettings are the account details a user can change themselves.
type UserSettings struct {
	Name              string `json:"name"`
	Currency          string `json:"currency"`
	DepositReturnDays int    `json:"deposit_return_days"`
	TimeZone          string `json:"time_zone"`
}

func (s *Store) UpdateUser(ctx context.Context, id string, in UserSettings) error {
	return execOne(s.pool.Exec(ctx, `
		UPDATE users SET name = $2, currency = $3, deposit_return_days = $4, time_zone = $5, updated_at = now()
		 WHERE id = $1`,
		id, in.Name, in.Currency, in.DepositReturnDays, in.TimeZone))
}

// SetPassword changes the password and bumps the token version, which signs
// out every session, and uses up any open password reset links. It returns
// the new version.
func (s *Store) SetPassword(ctx context.Context, id, passwordHash string) (int, error) {
	var version int
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			UPDATE users SET password_hash = $2, token_version = token_version + 1, updated_at = now()
			 WHERE id = $1
			RETURNING token_version`, id, passwordHash).Scan(&version)
		if err != nil {
			return notFound(err)
		}
		_, err = tx.Exec(ctx, `UPDATE password_resets SET used_at = now() WHERE user_id = $1 AND used_at IS NULL`, id)
		return err
	})
	return version, err
}
