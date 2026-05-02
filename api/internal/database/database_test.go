package database_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorin-work/estates/api/internal/database"
	"github.com/sorin-work/estates/api/internal/testdb"
)

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func count(t *testing.T, pool *pgxpool.Pool, sql string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func TestDepositChargesMoveOutOfRent(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := database.MigrateTo(ctx, pool, 10); err != nil {
		t.Fatal(err)
	}

	exec(t, pool, `INSERT INTO users (id, email, name, password_hash) VALUES ('00000000-0000-0000-0000-000000000001', 'o@example.com', 'O', 'x')`)
	exec(t, pool, `INSERT INTO properties (id, owner_id, name, address_line1, city) VALUES ('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-000000000001', 'P', '1 St', 'C')`)
	exec(t, pool, `INSERT INTO units (id, property_id, label) VALUES ('00000000-0000-0000-0000-0000000000b1', '00000000-0000-0000-0000-0000000000a1', '1'), ('00000000-0000-0000-0000-0000000000b2', '00000000-0000-0000-0000-0000000000a1', '2')`)
	exec(t, pool, `INSERT INTO tenants (id, owner_id, first_name, last_name) VALUES ('00000000-0000-0000-0000-0000000000c1', '00000000-0000-0000-0000-000000000001', 'T', 'One')`)
	exec(t, pool, `INSERT INTO leases (id, unit_id, tenant_id, start_date, end_date, rent_cents, deposit_cents) VALUES
		('00000000-0000-0000-0000-0000000000d1', '00000000-0000-0000-0000-0000000000b1', '00000000-0000-0000-0000-0000000000c1', '2026-01-01', '2026-12-31', 2000, 1000),
		('00000000-0000-0000-0000-0000000000d2', '00000000-0000-0000-0000-0000000000b2', '00000000-0000-0000-0000-0000000000c1', '2026-01-01', '2026-12-31', 2000, 500)`)
	// Lease 1 paid its deposit three days early, then January's rent of the
	// same amount. Lease 2 never paid its deposit.
	exec(t, pool, `INSERT INTO charges (lease_id, kind, amount_cents, due_date, description) VALUES
		('00000000-0000-0000-0000-0000000000d1', 'deposit', 1000, '2026-01-01', 'Security deposit'),
		('00000000-0000-0000-0000-0000000000d1', 'rent', 1000, '2026-02-01', 'Rent'),
		('00000000-0000-0000-0000-0000000000d2', 'deposit', 500, '2026-01-01', 'Security deposit')`)
	exec(t, pool, `INSERT INTO payments (lease_id, amount_cents, paid_on, notes) VALUES
		('00000000-0000-0000-0000-0000000000d1', 1000, '2025-12-29', 'Deposit'),
		('00000000-0000-0000-0000-0000000000d1', 1000, '2026-02-01', '')`)

	if err := database.MigrateTo(ctx, pool, 11); err != nil {
		t.Fatal(err)
	}

	var occurred, notes string
	err := pool.QueryRow(ctx, `SELECT occurred_on::text, notes FROM deposit_transactions WHERE lease_id = '00000000-0000-0000-0000-0000000000d1' AND kind = 'received' AND amount_cents = 1000`).Scan(&occurred, &notes)
	if err != nil {
		t.Fatalf("moved deposit: %v", err)
	}
	if occurred != "2025-12-29" || notes != "Deposit" {
		t.Errorf("deposit received %s (%q)", occurred, notes)
	}
	if n := count(t, pool, `SELECT count(*) FROM payments`); n != 1 {
		t.Errorf("%d payments left, want only the rent", n)
	}
	if n := count(t, pool, `SELECT count(*) FROM deposit_transactions`); n != 1 {
		t.Errorf("%d deposit transactions", n)
	}
	if n := count(t, pool, `SELECT count(*) FROM charges WHERE kind = 'deposit'`); n != 1 {
		t.Errorf("the unpaid deposit charge should stay, %d left", n)
	}
}
