# estates

A small property management app for landlords with a handful of buildings:
properties and units, tenants, leases, rent and payments, maintenance and
expenses, with a dashboard that shows what needs attention.

## Features

- **Properties and units**: houses, duplexes, apartment buildings and condos,
  each with its units (beds, baths, size, market rent) and live occupancy.
- **Tenants**: contact details, notes and lease history, with search.
- **Leases**: term, rent, deposit, due day, grace period and late fee. A unit
  cannot be let twice for the same dates. Leases can be ended early.
- **Rent**: monthly rent charges are generated automatically from each lease,
  with the first and last partial months prorated by day. Payments are applied
  to the oldest charges first, so every lease has a clear balance and
  "overdue since" date. Late fees are added once per missed month after the
  grace period.
- **Rent roll**: every active lease with its balance, overdue amount and how
  long it has been late, plus all payments received in a month.
- **Maintenance**: requests per property or unit with a priority and a status
  workflow (open → in progress / on hold → completed or cancelled, closed
  requests can be reopened). Expenses can be linked to a request to track
  its cost.
- **Expenses**: per property, by category (repairs, taxes, insurance,
  mortgage, utilities, HOA...), with totals for any period.
- **Dashboard**: occupancy, rent collected vs charged this month, overdue
  rent, leases ending in the next 60 days, open maintenance and a 12 month
  income vs expenses chart.

## Stack

| Part     | Tech                                                                     |
| -------- | ------------------------------------------------------------------------ |
| API      | Go 1.26, chi, PostgreSQL via pgx, goose migrations (embedded), JWT, slog |
| Web      | React 19, TypeScript, Vite, React Router, TanStack Query                 |
| Tests    | Go `testing` + httptest, Vitest + Testing Library                        |
| Tooling  | golangci-lint, ESLint, Prettier, GitHub Actions                          |
| Delivery | Docker Compose: Postgres 17, the API, and nginx serving the web app      |

## Running with Docker

```bash
cp .env.example .env   # optional, the defaults work
docker compose up --build
```

Then open http://localhost:5175 and sign in with the demo account:

- **Email:** `demo@example.com`
- **Password:** `demo-password`

The demo landlord is created on first start with four properties, ten units,
eleven tenants, a year of rent payments (including a couple of tenants who
are behind), maintenance requests and expenses. Set `SEED_DEMO=false` to
start with an empty database.

| Service | Port (host)           | Notes                                     |
| ------- | --------------------- | ----------------------------------------- |
| web     | http://localhost:5175 | nginx, proxies `/api` to the API          |
| api     | http://localhost:4200 | `GET /healthz` for health checks          |
| db      | `localhost:5434`      | Postgres 17, user/password/db `estates`   |

The ports can be changed with `WEB_PORT`, `API_PORT` and `DB_PORT`. The API
applies migrations when it starts. Data lives in the `db-data` volume;
`docker compose down -v` removes it.

## Running locally

You need Go 1.26+, Node 22+ and a Postgres database. The easiest is to run
only the database in Docker:

```bash
docker compose up -d db
```

API (listens on :4200):

```bash
cd api
cp .env.example .env
set -a; . ./.env; set +a
go run ./cmd/api
```

Web (Vite on :5173, proxies `/api` to localhost:4200):

```bash
cd web
npm install
npm run dev
```

## Tests and checks

```bash
# API: unit tests always run; database tests run when TEST_DATABASE_URL is set.
cd api
go vet ./...
golangci-lint run
TEST_DATABASE_URL=postgres://estates:estates@localhost:5434/estates?sslmode=disable go test ./...

# Web
cd web
npm run lint
npm run typecheck
npm test
```

Database tests create a throwaway schema per test and drop it afterwards, so
they can share the development database.

## Project layout

```
api/
  cmd/api/              entry point: config, migrations, demo seed, HTTP server
  internal/
    auth/               bcrypt passwords and JWT tokens
    billing/            rent and late fee generation, lease balances
    config/             environment configuration
    database/           pgx pool and embedded SQL migrations
    domain/             dates, rent schedules, ledgers, lease and maintenance rules
    httpapi/            router, middleware and JSON handlers
    seed/               demo data
    store/              Postgres queries, scoped to the owning user
    testdb/             per-test Postgres schemas
web/
  src/
    api/                typed API client
    auth/               auth context and route guards
    components/         shared UI and forms
    lib/                money, date and label helpers
    pages/              one component per route
```

## API conventions

- All endpoints except `/api/auth/*` and `/healthz` need an
  `Authorization: Bearer <token>` header. Tokens last 7 days.
- **Money** is integer cents (`145000` is $1,450.00). **Dates** are
  `YYYY-MM-DD` with no timezone.
- Errors look like
  `{"error": {"code": "validation_failed", "message": "...", "fields": {"rent_cents": "must be more than zero"}}}`.

## How rent works

Rent is due on the lease's due day (31 means the last day of the month). The
first charge is due on the start date; if the lease starts between due dates,
that first charge covers only the days until the next due date. A background
job runs every hour (`BILLING_INTERVAL`) and creates any charges that have
fallen due; it is idempotent, so running it again changes nothing. "Update
charges now" on the Rent page runs it immediately.

A charge is overdue once its grace period has passed: rent due on the 1st
with 5 days grace is overdue from the 7th. If the lease has a late fee, one
fee is added per overdue rent charge. Late fees are only added for rent that
became overdue in the last 30 days, so entering an existing lease with its
history does not pile up fees for months whose payments have not been typed
in yet.
