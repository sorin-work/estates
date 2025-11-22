// Package store is the Postgres data access layer. Every query that touches
// user data is scoped to the owning user, so a handler can never read or
// change another landlord's records by guessing an ID.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNotFound means the record does not exist or belongs to someone else.
	ErrNotFound = errors.New("not found")
	// ErrConflict means the change clashes with existing data.
	ErrConflict = errors.New("conflict")
)

// ConflictError wraps ErrConflict with a message safe to show to users.
type ConflictError struct{ Msg string }

func (e *ConflictError) Error() string { return e.Msg }
func (e *ConflictError) Unwrap() error { return ErrConflict }

func conflict(msg string) error { return &ConflictError{Msg: msg} }

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Ping checks the database connection.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, s.pool, fn)
}

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func isUniqueViolation(err error) bool     { return pgCode(err) == "23505" }
func isForeignKeyViolation(err error) bool { return pgCode(err) == "23503" }

// notFound maps pgx.ErrNoRows to ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// collectOne scans exactly one row into T by column name.
func collectOne[T any](rows pgx.Rows, err error) (T, error) {
	if err != nil {
		var zero T
		return zero, err
	}
	v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[T])
	return v, notFound(err)
}

// collectAll scans all rows into a slice of T by column name. It never
// returns a nil slice, so empty lists encode as [] rather than null.
func collectAll[T any](rows pgx.Rows, err error) ([]T, error) {
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []T{}
	}
	return out, nil
}

// execOne runs a statement that must affect exactly one row.
func execOne(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
