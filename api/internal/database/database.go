// Package database opens the Postgres pool and applies schema migrations.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Connect opens a connection pool and waits for the database to accept
// connections, retrying for up to the context deadline.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	for attempt := 1; ; attempt++ {
		err = pool.Ping(ctx)
		if err == nil {
			return pool, nil
		}
		slog.Warn("database not ready", "attempt", attempt, "err", err)
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, fmt.Errorf("connect to database: %w", err)
		case <-time.After(time.Second):
		}
	}
}

// Migrate applies all pending migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	return migrate(ctx, db)
}

func migrate(ctx context.Context, db *sql.DB) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, mustSub(migrations, "migrations"))
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		slog.Info("applied migration", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}
	return nil
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
