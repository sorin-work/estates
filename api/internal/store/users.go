package store

import (
	"context"
	"time"
)

type User struct {
	ID           string    `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	Name         string    `db:"name" json:"name"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Currency     string    `db:"currency" json:"currency"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

const userColumns = `id, email, name, password_hash, currency, created_at`

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
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

func (s *Store) UpdateUser(ctx context.Context, id string, in UserSettings) error {
	return execOne(s.pool.Exec(ctx, `
		UPDATE users SET name = $2, currency = $3, updated_at = now() WHERE id = $1`,
		id, in.Name, in.Currency))
}

func (s *Store) SetPassword(ctx context.Context, id, passwordHash string) error {
	return execOne(s.pool.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, passwordHash))
}
