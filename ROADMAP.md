# Roadmap

Things I might add to estates. The app covers the day to day of a small
landlord today (units, leases, rent, deposits, maintenance, expenses and a
profit and loss report), so nothing here is promised or scheduled. These are
ideas, written down with enough detail that I (or anyone else) can pick one
up later without starting from zero. Items may change, move between groups
or be dropped.

What the app does today is in the [README](README.md) and the
[API reference](docs/api.md).

- **Next**: small or well understood. I would start with these.
- **Later**: worth doing, but bigger or waiting on a decision.
- **Maybe**: not sure yet that they are worth the extra complexity.

Sizes are rough: **S** is an evening or two, **M** a few evenings up to a
couple of weekends, **L** several weeks of spare time.

Each item is meant to fit in one pull request and to be covered by the test
suite. Where the tests and the logic go:

- Rules that can be worked out without a database (dates, rent schedules,
  ledgers, reminders, recurrences) belong in `api/internal/domain` as plain
  functions with table tests next to them. A handler or a query should not
  be the only place a rule lives.
- Queries live in `api/internal/store`, always scoped to the owner. They are
  tested in `store_test.go` against a throwaway Postgres schema
  (`internal/testdb`), which runs when `TEST_DATABASE_URL` is set.
- Handlers in `api/internal/httpapi` are tested through the real router with
  `httptest`, mostly against Postgres (`api_test.go`, `handlers_test.go` and
  one file per feature). Every new endpoint needs a test that another
  landlord's ID answers `404`.
- Each schema change is a new numbered goose migration under
  `api/internal/database/migrations`, with a working `Down`.
- The web app has no browser tests. Pages and components are tested with
  Vitest and Testing Library next to the file (`RentPage.test.tsx`), with
  `mockFetch` from `src/test/render.tsx` standing in for the API. Pure
  helpers go in `web/src/lib` with their own tests.

File names below are relative to `api/internal/` for Go and `web/src/` for
the web app unless they start with `api/`, `web/`, `docs/` or the repository
root.

"Depends on" lists real prerequisites only. Items that merely touch the same
file are not listed there; the bigger overlaps are mentioned in the items.

When an item ships, its Group cell in the Overview becomes `Shipped`, its
section moves under a `## Shipped` heading at the end with one paragraph
(what was built, how it differs from the plan, what is still open), and the
README, `docs/` and `CHANGELOG.md` (newest changes under an `Unreleased`
heading at the top) are updated in the same pull request.

## Overview

| #   | Item                                         | Group | Area          | Size | Depends on |
| --- | -------------------------------------------- | ----- | ------------- | ---- | ---------- |
| 1   | Slow down password guessing, close sign-up   | Next  | security      | S    | -          |
| 2   | Security headers                             | Next  | security      | S    | -          |
| 3   | The landlord's time zone decides "today"     | Next  | rent          | M    | -          |
| 4   | Scheduled backups and a restore guide        | Next  | ops           | S    | -          |
| 5   | Tenant statements                            | Next  | rent          | S    | -          |
| 6   | Receipts and invoices on expenses            | Later | expenses      | S    | -          |
| 7   | Recurring extra charges on a lease           | Later | rent          | M    | -          |
| 8   | Refunds and tenant credit                    | Later | rent          | M    | -          |
| 9   | Mortgage interest and principal              | Later | reports       | M    | -          |
| 10  | Year-end tax summary                         | Later | reports       | S    | 9          |
| 11  | What each vendor was paid in a year          | Later | vendors       | S    | -          |
| 12  | Vacancy and lost rent                        | Later | reports       | S    | -          |
| 13  | Calendar feed                                | Later | dashboard     | S    | -          |
| 14  | Import payments from a bank export           | Later | rent          | M    | -          |
| 15  | Import a portfolio from spreadsheets         | Later | import        | M    | -          |
| 16  | Export everything, delete the account        | Later | account       | M    | -          |
| 17  | Signed-in devices and signing them out       | Later | security      | M    | -          |
| 18  | Cookie sessions instead of a stored token    | Later | security      | M    | 17         |
| 19  | Two-step sign-in with an authenticator app   | Later | security      | M    | -          |
| 20  | Email delivery and a weekly digest           | Later | notifications | M    | -          |
| 21  | Password reset by email                      | Later | account       | S    | 20         |
| 22  | Send receipts and statements to tenants      | Later | notifications | M    | 5, 20      |
| 23  | Several tenants on one lease                 | Later | leases        | L    | -          |
| 24  | Updates and notes on maintenance requests    | Later | maintenance   | M    | -          |
| 25  | Move-in and move-out inspections             | Later | leases        | L    | -          |
| 26  | Printable notices                            | Later | leases        | S    | -          |
| 27  | Pages and server filters for long lists      | Later | performance   | M    | -          |
| 28  | Billing status in Settings                   | Later | ops           | S    | -          |
| 29  | Readiness check and request metrics          | Later | ops           | S    | -          |
| 30  | Migration and upgrade checks in CI           | Later | tests         | S    | -          |
| 31  | Dashboard and report numbers pinned by tests | Later | tests         | S    | -          |
| 32  | Accessibility checks in the test suite       | Later | ui            | M    | -          |
| 33  | Browser tests for the main flows             | Maybe | tests         | M    | -          |
| 34  | Shared access for a partner or a manager     | Maybe | account       | L    | 17         |
| 35  | Tenant portal                                | Maybe | tenants       | L    | 20, 34     |
| 36  | Online rent payments                         | Maybe | rent          | L    | 35         |
| 37  | Balances worked out in SQL                   | Maybe | performance   | L    | 27         |
| 38  | Interest on deposits                         | Maybe | deposits      | S    | -          |

## Next

### 1. Slow down password guessing, close sign-up

`POST /api/auth/login` answers as fast as bcrypt allows, for as long as
anyone keeps asking. Nothing counts failed attempts, so a script can try a
password list against `demo@example.com` or a real account for days. And
`POST /api/auth/register` is open to anyone who can reach the server: a
landlord who puts the app on the internet for themselves also hosts it for
strangers.

- A small in-memory limiter in a new `httpapi/ratelimit.go`: a token bucket
  per key with a clock the tests can replace, and a sweep that forgets idle
  keys so the map does not grow forever.
- Two keys on login: the client address (say 20 attempts a minute) and the
  lower-cased email (say 5 failures in 15 minutes). Only failures count
  against the email, so the owner signing in normally never trips it. A
  success clears the email's count.
- Over the limit the answer is `429` with code `too_many_attempts` and a
  `Retry-After` header. The login page shows "Too many attempts, try again
  in N minutes" instead of the generic error.
- Register gets the address limit too, so nobody can create accounts in a
  loop.
- A new `ALLOW_REGISTRATION` setting (default `true`, so the demo and the
  local setup keep working). When it is `false`, register answers `403`
  with code `registration_closed`, unless there are no users at all, so a
  fresh install can still create its first account. `GET /api/auth/config`
  (no token needed) tells the web app whether to show the "Create an
  account" link and route.
- Tricky: the client address. Behind the nginx in `web/nginx.conf` every
  request comes from the web container, so the limit would be shared by
  everyone. nginx already sets `X-Forwarded-For`. A `TRUSTED_PROXIES`
  setting (a list of CIDRs, empty by default) decides when that header is
  believed; trusting it always would let a script pick a new address per
  request. Docker Compose sets it to the compose network.
- Tricky: the timing defence in `login` (`dummyHash`) must still run for
  unknown emails, and a limited email must answer the same way whether or
  not the account exists, or the limiter becomes a way to find accounts.
- Undecided: whether the limiter should also lock the account for a while.
  I lean towards no. It hands anyone a way to lock the owner out, and the
  per-email bucket already makes guessing useless.
- Tests: unit tests for the bucket with a fake clock; router tests for 429
  and `Retry-After`, the count reset on success, the closed sign-up with and
  without existing users, and the forwarded header only trusted from a
  listed proxy. A Vitest case for the login page message.

### 2. Security headers

The API sets `X-Content-Type-Options` only on document downloads, and the
nginx in `web/nginx.conf` sets nothing. The app keeps its login token in
`localStorage` (`api/client.ts`), so a single injected script would own the
account, and nothing tells the browser to refuse inline or third-party
scripts. Nothing stops another site from framing the app either.

- nginx, for the app shell: a `Content-Security-Policy` of `default-src
'self'`, `script-src 'self'`, `style-src 'self'`, `img-src 'self' data:`,
  `connect-src 'self'`, `frame-ancestors 'none'`, `base-uri 'self'`,
  `form-action 'self'`, `object-src 'none'`. Plus `X-Content-Type-Options:
nosniff`, `Referrer-Policy: same-origin` and a `Permissions-Policy` that
  turns off camera, microphone and location.
- The API gets a middleware that sets `nosniff`, `Referrer-Policy` and a
  strict CSP (`default-src 'none'; frame-ancestors 'none'`) on every
  response, so JSON opened directly in a tab cannot run anything either.
  JSON and CSV responses also get `Cache-Control: no-store`, since they
  carry tenants' details.
- `Strict-Transport-Security` only makes sense behind HTTPS, which this
  setup does not terminate. Document it in the README for whoever puts a
  TLS proxy in front, rather than sending it over plain HTTP.
- Tricky: inline code. `web/index.html` loads only the bundle today, and
  the theme is applied from `lib/theme.ts` inside it. A later "apply the
  theme before first paint" snippet in `index.html` would be blocked, so
  it has to stay a file. Inline `style` attributes set from React are
  allowed by CSP, but a `<style>` block is not. Vite's dev server is not
  behind nginx, so check the production build with `npm run preview`.
- Tricky: `add_header` in nginx does not inherit into a `location` that has
  its own `add_header`. The `/assets/` block sets `Cache-Control`, so the
  headers need to be repeated there or moved into a shared include.
- Undecided: a CSP report endpoint. Not for now; nobody would read the
  reports.
- Tests: a router test that a JSON response, a CSV export and an error all
  carry the headers. The nginx side has no tests; the docker job in CI can
  `curl -I` the shell and check for the CSP header.

### 3. The landlord's time zone decides "today"

`domain.Today()` is the current date in UTC, and so is the clock behind
every handler (`Server.today`) and the billing loop (`billing.Run`). For a
landlord in California the new month starts at 4pm on the last day: rent
for tomorrow is charged, "overdue since" moves and the dashboard's "this
month" flips while it is still today for them. In Sydney or Auckland the
opposite happens in the morning. Late fees are added a day early or late in
the same way.

- Data: `time_zone text NOT NULL DEFAULT 'UTC'` on `users`, an IANA name.
  `PUT /api/auth/me` takes it, validated with `time.LoadLocation`. The
  settings page offers the browser's zone
  (`Intl.DateTimeFormat().resolvedOptions().timeZone`) as the default and a
  searchable list from `Intl.supportedValuesOf('timeZone')`.
- `domain.TodayIn(loc)` next to `Today`. `Server.today` becomes
  `func(r *http.Request) domain.Date`, or a helper that reads the user's
  zone once per request; the forty or so call sites change mechanically.
  Tests keep replacing the clock.
- Billing: `BillAll` uses one `asOf` for every lease. It needs the date per
  owner. `BillableLeases` returns the owner's zone with each lease (one more
  join it already has the tables for), and the loop works out the date for
  each zone once per run.
- `api/Dockerfile` already installs `tzdata`. For local runs without it,
  import `time/tzdata` in `cmd/api` so the binary carries the zone
  database.
- Tricky: existing accounts. `UTC` keeps today's behaviour, so nothing
  moves until the landlord picks a zone. The settings page can point out
  that the browser's zone differs from the saved one.
- Tricky: the billing interval is an hour, so a charge can still arrive up
  to an hour after local midnight. That is fine, but the README should stop
  implying it happens at midnight.
- Tricky: the dates stored in the database (`created_at` for activity) are
  timestamps and display fine already; only the "what day is it" decisions
  change. Payment and expense dates typed by the user are plain dates and
  are not touched.
- Undecided: a zone per property instead of per landlord. Someone with
  buildings in two zones exists, but rent is still one person's to-do
  list. I lean towards per landlord.
- Tests: domain tests for `TodayIn` around midnight; a billing test with
  two owners in different zones billed in the same run; a router test that
  the dashboard month follows the user's zone; Vitest for the settings
  field.

### 4. Scheduled backups and a restore guide

The README shows one `pg_dump` command and stops there. Everything lives in
the database, the uploaded documents included (`documents.content`), so a
lost volume loses the leases, the payment history and the signed PDFs at
once. There is no schedule, no retention and no written way back.

- A `backup` service in `docker-compose.yml` on the same
  `postgres:17-alpine` image, running a small shell script from a new
  `scripts/backup.sh`: `pg_dump -Fc` into a `backups` bind mount every
  `BACKUP_INTERVAL` (default a day), named by UTC timestamp, keeping the
  newest `BACKUP_KEEP` (default 14). It writes to a temporary name and
  renames, so a half-written dump never looks like a good one.
- A `docs/backups.md` with how to restore into a fresh volume
  (`pg_restore --clean --if-exists`), how to check a dump without
  restoring (`pg_restore --list`), and how to copy backups off the machine.
  A volume on the same disk is not a backup on its own, and the guide
  should say so.
- An upgrade section in the same guide: take a backup, pull, `docker
compose up --build -d`; the API applies migrations on start. Going back
  means restoring the backup, because migrations are only tested upwards.
- `make backup` and `make restore FILE=...` shortcuts.
- Tricky: a dump can be large because of the documents. Custom format
  compresses, but PDFs and photos barely shrink. Say so, and keep the
  retention in files, not in days, so it is predictable.
- Tricky: the backup container must not start before the database is
  healthy, and must keep going if one run fails (log and wait for the next
  one) rather than exit and restart in a loop.
- Undecided: shipping dumps to object storage. Not in this item; the guide
  can show `rclone` or `aws s3 cp` as an example.
- Tests: the script is shell, so the check is the docker job: run one
  backup after the stack is up and restore it into a second database with
  `pg_restore`. If that is too slow for CI, a `scripts/backup-test.sh` that
  does the same locally.

### 5. Tenant statements

A tenant who asks "what do I owe and how did you get there" gets either a
screenshot of the lease page or a pile of receipts. The ledger on
`GET /api/leases/{id}/ledger` has everything, but only from the start of the
lease and with no running balance, and there is no printable version. Housing
benefit offices and tenants moving out often ask for exactly this document.

- A pure `domain.BuildStatement(charges, payments, from, to)` that returns
  the opening balance (charges due before `from` less payments before
  `from`), every charge and payment in the period in date order with the
  running balance after each line, and the closing balance. Charges that
  fall due after `to` are left out, payments made in advance are not.
- `GET /api/leases/{id}/statement?from=&to=` (default: the lease start to
  today), using the same `LedgerEntries` query as the ledger, plus the
  landlord, tenant and property details the receipt already loads in
  `httpapi/receipts.go`.
- A new `pages/StatementPage.tsx` at `/leases/:id/statement`, outside the
  layout like `ReceiptPage`, with a period picker, Print, and the same print
  styles. A "Statement" button on the lease page.
- Tricky: deposit charges. Old leases may still have `deposit` charges from
  before deposits were tracked apart (migration 00011). The statement should
  treat charges the way the ledger does, so both show the same balance.
- Tricky: payments on the same day as a charge. Show the charge first, so
  the running balance does not dip below zero for a moment.
- Tricky: a renewed lease is a separate lease. The statement covers one
  lease; the renewal links on the lease page lead to the next statement.
- Undecided: a CSV version. Cheap to add next to the print view; I would
  wait until someone asks.
- Tests: domain tests for opening balance, a period in the middle of the
  lease, prepayment and an empty period; a router test including another
  landlord's lease; a Vitest test for the page with the running balance.

## Later

### 6. Receipts and invoices on expenses

Documents can be attached to a tenant, a lease or a property
(`num_nonnulls(tenant_id, lease_id, property_id) = 1` in migration 00017),
but not to an expense. The plumber's invoice and the insurance bill are the
papers a bookkeeper or a tax office asks for, and today they end up on the
property with no link to the amount.

- Data: `expense_id uuid REFERENCES expenses ON DELETE CASCADE` on
  `documents`, and the check widened to four columns. `DocumentTarget` and
  `documentTarget` in `httpapi/documents.go` take `expense_id`.
- The expense form gets the `Documents` component, and the expenses list a
  paperclip with the count, so missing receipts are easy to spot.
- The expenses CSV export gets a "receipt" column (yes or no).
- Tricky: recurring expenses create a new expense each time; an attachment
  belongs to one occurrence, not the recurring setup.
- Tricky: an expense on a maintenance request. Show its receipts on the
  request too, read-only, rather than attaching them twice.
- Tests: store and router tests for upload, list and delete on an expense,
  someone else's expense is 404, deleting the expense removes the file.

### 7. Recurring extra charges on a lease

Only rent is generated. Parking, a storage locker, pet rent or a flat
utility fee have to be added by hand every month through
`POST /api/leases/{id}/charges`, and a forgotten one is money lost. Recurring
expenses already solve the same problem on the other side of the books.

- Data: `lease_charges` (lease, kind `utility | other`, description,
  amount, from and until dates) and `charges.lease_charge_id` with a unique
  index on `(lease_charge_id, due_date)`, the same trick as
  `expenses_one_per_occurrence`.
- They fall due on the lease's due day with the rent and stop with the
  lease. `domain.RentSchedule` gets a sibling that produces the extra
  charges for the same periods, prorated the same way.
- `billing.BillLease` adds them next to rent. A late fee applies to rent
  only, as today.
- The lease form and lease page list them; a renewal copies them to the
  new lease.
- Tricky: ending a lease early or changing its dates must remove or
  reprorate these charges the way it does rent (`UpdateLease` already
  rewrites rent charges). One code path for both would be best.
- Tricky: deleting a charge by hand. Rent charges cannot be deleted
  (`fix(api): rent charges cannot be deleted by hand`); generated extras
  should follow the same rule, or the next run brings them back.
- Tests: domain tests for the schedule and proration, billing tests for
  idempotence, router tests for create, end early and renewal.

### 8. Refunds and tenant credit

Payments must be more than zero (`CHECK (amount_cents > 0)`), so money that
goes back to the tenant has nowhere to go. An overpayment shows as a
negative balance forever, and a refunded overpayment can only be recorded
by deleting or editing the original payment, which breaks the receipt
numbers.

- A `refund` payment kind: `kind text NOT NULL DEFAULT 'payment'` on
  `payments`, amount still positive, subtracted in `BuildLedger`. Refunds
  get no receipt number.
- The rent roll and the lease page label a negative balance "In credit".
  The lease page offers "Refund credit" with the amount filled in.
- When a lease ends in credit, the deposit return reminder mentions the
  credit too.
- Tricky: the profit and loss report counts payments as collected; a
  refund must reduce it in the month it is paid.
- Tricky: oldest-first allocation. A refund should come off the pool before
  charges are paid, which keeps `BuildLedger` simple; check that a refund
  never makes a paid charge look unpaid unless the money really went back.
- Undecided: moving credit to a renewal lease. The deposit moves already;
  I lean towards doing the same for credit, as a pair of entries.
- Tests: ledger unit tests with refunds, report totals, receipt numbers
  unaffected, router validation.

### 9. Mortgage interest and principal

`mortgage` is an expense category, and the profit and loss report
subtracts the whole payment. Only the interest is a cost; the principal is
the landlord paying themselves back. A property with a mortgage looks far
less profitable than it is, and the numbers cannot go to an accountant as
they are.

- A recurring mortgage expense can carry the loan: principal at start,
  annual rate and term, or simply the split typed in per payment. A pure
  `domain.Amortize` produces the interest and principal of each payment.
- Data: `mortgage_principal_cents` on `expenses` (zero for everything
  else). The report subtracts only the interest and shows principal paid on
  its own line below the net.
- Tricky: existing mortgage expenses have no split. Show them as they are,
  with a note on the report that they include principal.
- Tricky: extra payments and rate changes break a computed schedule. The
  split per payment should be editable after the fact.
- Tests: amortisation tables against known values, report totals with and
  without the split.

### 10. Year-end tax summary

The bookkeeper gets three CSVs (report, payments, expenses) and builds the
year-end summary by hand. In the US that is Schedule E: rents received and
expenses by fixed lines (advertising, cleaning, insurance, legal,
management, mortgage interest, repairs, taxes, utilities...).

- `GET /api/reports/tax-summary?year=` per property: rent received,
  withheld deposits, expenses by category, mortgage interest from item 9.
- A mapping from `ExpenseCategories` to report lines in `domain`, and a
  printable page next to the profit and loss report.
- Tricky: this is not tax advice and every country differs. Label it a
  summary by category, keep the US lines as one mapping, and do not invent
  others.
- Depends on item 9, since mortgage interest is one of the biggest lines.
- Tests: domain test for the mapping, router test for a year with data in
  two properties.

### 11. What each vendor was paid in a year

US landlords send a 1099-NEC to contractors paid $600 or more in a year,
and need the vendor's tax ID and address to do it. The vendor page shows the
total spent, but over all time and without those details.

- `tax_id` and `address` on `vendors`, and a `1099` flag that defaults to
  true for trades and false for utilities and companies.
- `GET /api/vendors/payments?year=` with totals per vendor and a CSV export,
  flagging those over a threshold set in Settings.
- Tricky: expenses typed with a free-text vendor name and no `vendor_id`
  are missed. Link them first (the vendor import already does this for
  names that match).
- Tricky: tax IDs are sensitive. Mask them in lists and the activity log.
- Tests: store totals by year, CSV export, masking.

### 12. Vacancy and lost rent

The property page shows how long a unit has been empty, but not what that
cost. "Vacancy cost me $4,200 last year" is the number that decides whether
to lower the asking rent.

- A pure function that, for a unit and a period, finds the days not
  covered by any lease and values them at the unit's market rent
  (`market_rent_cents`), or the last lease's rent when there is none.
- A section on the profit and loss report per property, and a vacancy rate.
- Tricky: a unit added halfway through the period was not vacant before it
  existed; use `units.created_at` as the earliest date.
- Tests: domain tests for gaps, overlaps with terminated leases and leap
  years.

### 13. Calendar feed

Lease ends, move-ins, rent increases and deposit deadlines are on the
dashboard, but people live in their phone's calendar.

- `GET /api/calendar/{token}.ics`: a secret token per user in the URL,
  since calendar apps cannot send a bearer header. Settings shows the URL
  with "Copy" and "Make a new link" (which retires the old token).
- Events come from the same facts as `domain.Reminders`, as all-day events
  with stable `UID`s so updates replace rather than duplicate.
- Tricky: the token is a password. Store a hash, compare in constant time,
  rate limit, and keep the feed to what is needed (no amounts, first names
  only, or a setting for it).
- Tests: the iCalendar writer as a unit test (line folding, escaping);
  router tests for a bad token and a rotated token.

### 14. Import payments from a bank export

Recording twelve payments a month by hand is the most repetitive part of
using the app. Every bank can export a CSV.

- Upload a CSV, map its columns once (date, amount, description), and
  remember the mapping per account.
- A pure matcher in `domain` suggests a lease for each incoming amount: the
  tenant's name or lease reference in the description, then the exact
  amount due. The page shows a preview where each line is confirmed,
  changed or skipped; nothing is saved until "Record payments".
- Duplicates: a hash of date, amount and description is kept on the
  payment (`import_ref`), so the same file twice adds nothing.
- Tricky: date and number formats (`1.450,00`, `14/10/2026`). Ask on the
  mapping step instead of guessing.
- Tricky: outgoing lines and lines that are not rent. Skip negatives by
  default.
- Tests: CSV parsing and matching as domain tests, router tests for the
  preview and the commit, the repeat import.

### 15. Import a portfolio from spreadsheets

A landlord with eight units who wants to try the app has to type every
property, unit, tenant and lease into forms first. Most of them have it in
a spreadsheet already.

- One CSV template per kind (properties with units, tenants, leases),
  downloadable from an "Import" page, uploaded in that order.
- A dry run first: every row checked with the same validation the forms use
  (the validators in `httpapi/validate.go` and the store's overlap check),
  errors listed by row and column. Commit runs in one transaction.
- Leases import with their history: rent is charged back to the start as
  today, and late fees stay limited to the last 30 days
  (`LateFeeWindowDays`), so the opening balance can be entered as one
  payment.
- Tricky: references between files. Units by property name and label,
  leases by unit and tenant email. Say clearly which one did not match.
- Tests: domain tests for parsing, router tests for a dry run with errors
  and a clean commit.

### 16. Export everything, delete the account

There is no way to take your data out except the three CSVs, and no way to
close an account. A landlord leaving the app, or a self-hoster moving
servers without `pg_dump`, needs both. Under GDPR it is also the tenants'
data the landlord is responsible for.

- `GET /api/account/export`: a zip with one JSON file per table for the
  owner (properties, units, tenants, leases, charges, payments, deposits,
  maintenance, vendors, expenses, activity) and the documents in folders.
  Streamed, so a large account does not sit in memory.
- `DELETE /api/account` with the current password. Every table already
  cascades from `users`.
- Tricky: the export is the most sensitive request the API has. It needs
  the password again, not only the token.
- Undecided: importing such a zip into another server. Useful but large;
  not in this item.
- Tests: the zip contains every table and nothing from another owner;
  deleting leaves no rows behind (count every table).

### 17. Signed-in devices and signing them out

Tokens are JWTs that last seven days. Changing the password ends all of
them (`token_version`), but there is no way to see where you are signed in
or to end one session, and signing out in the web app only forgets the
token locally.

- Data: a `sessions` table (id, user, created, last seen, user agent,
  address). The token carries the session ID; `requireAuth` checks it
  exists, and updates `last_seen` at most once a minute.
- "Sign out" calls `DELETE /api/auth/session`. Settings lists sessions with
  "Sign out" per row and "Sign out everywhere else".
- Tricky: `requireAuth` already loads the user on every request. Fold the
  session check into that query rather than adding a second one.
- Tests: router tests for listing, ending one, ending the others, and a
  token for an ended session being refused.

### 18. Cookie sessions instead of a stored token

The token sits in `localStorage` (`api/client.ts`), where any script on the
page can read it. An `HttpOnly` cookie cannot be read by scripts at all.

- Login sets `HttpOnly; Secure; SameSite=Lax` cookie with the session from
  item 17; `requireAuth` accepts the cookie or the bearer header (for
  scripts and tests).
- CSRF: with `SameSite=Lax` and a JSON-only API most of it is covered;
  add a check that state-changing requests carry `Content-Type:
application/json` or a custom header, which a cross-site form cannot set.
  The multipart upload needs the custom header.
- Tricky: `Secure` cookies do not work over plain HTTP on anything but
  localhost. The local Docker setup is HTTP on localhost, which browsers
  allow; document it for LAN setups.
- Depends on item 17, so a cookie can be ended on the server.
- Tests: router tests for cookie auth, the CSRF check and logout clearing
  the cookie; Vitest that the client no longer stores a token.

### 19. Two-step sign-in with an authenticator app

A landlord's account holds tenants' names, phone numbers and ID copies.
A password alone is thin protection for that.

- TOTP (RFC 6238) written against the standard library (`crypto/hmac`,
  `crypto/sha1`): it is about forty lines and avoids a dependency.
- Settings: "Turn on two-step sign-in" shows a QR code (generated in the
  browser from the `otpauth://` URL; a small library or a plain text secret
  if a QR library is too heavy) and asks for a code to confirm. Ten
  one-time recovery codes, stored hashed.
- Login answers `{two_factor_required: true, challenge}` after the password;
  a second call with the code issues the token. The limiter from item 1
  covers both steps.
- Tricky: clock drift. Accept one step either side, and remember the last
  used step so a code cannot be replayed.
- Tests: RFC 6238 test vectors, router tests for the two-step login,
  recovery codes used once.

### 20. Email delivery and a weekly digest

The app never tells anyone anything; the landlord has to open it to find
out that rent is late. Several later items need email (password reset,
receipts to tenants).

- SMTP settings (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`,
  `MAIL_FROM`); without them email is off and the features that need it
  are hidden. `net/smtp` with STARTTLS is enough.
- An `outbox` table written in the same transaction as whatever triggers a
  message, and a worker that sends and retries with backoff. A crash never
  loses or doubles a message.
- First use: a weekly digest of `domain.Reminders` plus overdue rent, on a
  weekday chosen in Settings, in the landlord's time zone (item 3 makes
  that correct).
- Docker Compose gets a Mailpit service for local runs.
- Tests: the outbox worker against a fake sender (retries, no double
  send), digest content as a domain test.

### 21. Password reset by email

A forgotten password today means asking whoever runs the server to change
the hash in the database.

- "Forgot password" sends a link with a random token (stored hashed, 30
  minutes, one use). Setting a new password bumps `token_version`, so other
  sessions end.
- The answer is the same whether the email exists or not, and the limiter
  from item 1 applies.
- Depends on item 20.
- Tests: token expiry, one use, unknown email answers the same.

### 22. Send receipts and statements to tenants

Receipts (`ReceiptPage`) and statements (item 5) are printed or saved as
PDF and then emailed by hand.

- "Email to tenant" on the receipt and statement pages, with an option to
  send every receipt automatically when a payment is recorded.
- The email carries the receipt as HTML in the body; no PDF rendering on
  the server.
- Tricky: the tenant's email may be empty or wrong. Show where it went in
  the lease activity.
- Depends on items 5 and 20.

### 23. Several tenants on one lease

A lease has exactly one `tenant_id`. Couples and flatmates sign together,
and today the second person is a note.

- `lease_tenants (lease_id, tenant_id, is_primary)`, filled from
  `leases.tenant_id` by the migration. The primary tenant is who the
  receipt is addressed to.
- Every query that joins `leases.tenant_id` (rent roll, search, reminders,
  tenant history, receipts, seed) changes. That is why this is L.
- Tricky: payments stay per lease, not per person; a "paid by" field on the
  payment covers who sent it.
- Undecided: dropping `leases.tenant_id` or keeping it as the primary. I
  lean towards keeping it for one release and then dropping it.
- Tests: migration test, store tests for each changed query.

### 24. Updates and notes on maintenance requests

A request has a title, a description and a status, and that is all. "Called
the plumber, coming Thursday", "part on order" and "tenant not home" have
no place.

- `maintenance_updates (request, body, created_at)` with a timeline on the
  request, status changes appearing in it automatically.
- Photos through the existing documents, with `maintenance_request_id`
  added like `expense_id` in item 6.
- Tests: store and router tests, status change entries.

### 25. Move-in and move-out inspections

Deposit disputes are won with a dated move-in record. Today that is an
uploaded PDF from somewhere else.

- An inspection per lease (move-in, move-out) with rooms and items, each
  with a condition and notes, and photos through documents. A default room
  list per unit, editable.
- A side-by-side view of move-in and move-out, and a printable version for
  the tenant to sign.
- Tricky: photos from a phone are large. Resize in the browser before
  upload, or the 10 MB document limit and the database size become a
  problem.
- Tests: store tests for the structure, Vitest for the comparison view.

### 26. Printable notices

Rent increases and lease ends need a written notice, often with a minimum
notice period. The app knows the dates and amounts.

- Printable letters from the lease page: rent increase (from a rent
  change), notice of lease end, and a late rent reminder with the overdue
  amount from the ledger.
- Plain templates, outside the layout like `ReceiptPage`. No legal wording
  beyond the facts; a free text box for the rest.
- Tests: Vitest for each template with a fixed lease.

### 27. Pages and server filters for long lists

Payments page through results, but tenants, leases, expenses, maintenance
and vendors load the whole list (`ListTenants` has no limit) and the web app
filters some of them in the browser. That is fine for ten units and slow for
two hundred.

- `limit` and `offset` with a `total` on each list, reusing the helper in
  `store/util.go`, and `?q=` on leases and expenses.
- The pages move their filters into the request (they already keep them in
  the address with one hook) and show "Load more" or pages.
- Tricky: the expenses list returns totals by category. Those must cover
  the whole filter, not the page.
- Tests: store tests for paging and totals, page tests for "Load more".

### 28. Billing status in Settings

The billing loop logs failures and carries on (`BillAll` logs and
continues). Nobody reads the API's logs on a home server, so a lease that
fails to bill every hour goes unnoticed.

- A `billing_runs` table (started, finished, counts, failures with lease
  IDs) kept for 30 days. Settings shows the last run per owner and any
  lease that failed, with a link.
- Tests: billing test that a failing lease is recorded and others still
  run.

### 29. Readiness check and request metrics

`/healthz` pings the database. It says nothing about whether migrations
ran or whether billing is stuck, and there are no numbers to graph.

- `/healthz` stays a liveness check. `/readyz` adds the migration version
  and the age of the last billing run (item 28 if it exists, a timestamp in
  memory otherwise).
- `/metrics` in Prometheus text format, written by hand: requests by route
  and status, durations, billing runs. Off unless `METRICS_ADDR` is set, and
  served on that separate address so it is not public.
- Tests: router tests for both endpoints.

### 30. Migration and upgrade checks in CI

Every migration has a `Down`, but nothing runs them. Nothing checks that a
database at last release's version upgrades cleanly with data in it either;
the tests always start from an empty schema.

- A test in `database_test.go` that runs all migrations up, all down, and
  up again in a throwaway schema.
- A test that migrates to the version of the previous tag, seeds the demo
  data at that version (or a small fixture), then migrates to head and
  checks the data reads back.
- Tests: those two are the item.

### 31. Dashboard and report numbers pinned by tests

The dashboard and the profit and loss report are the numbers landlords
trust most, and they are mostly tested piece by piece. The demo seed
(`seed.Demo`) builds a full year of history with fixed dates.

- A store test that seeds the demo data with a fixed `today` and checks the
  dashboard and report totals against known values, so a query change that
  moves a total by a cent fails.
- Tricky: `seed.Demo` takes its dates from `domain.Today()`, so the totals
  move every day. It needs a `today` parameter (main passes the real one)
  before its numbers can be pinned.

### 32. Accessibility checks in the test suite

The forms, dialogs and skip link were made keyboard and screen reader
friendly by hand, with focused tests. Nothing catches a new page that
forgets a label.

- `vitest-axe` (or `axe-core` directly) in the page tests: one check per
  page with typical data, failing on serious violations.
- Fix what it finds, starting with charts (`components/charts`), which need
  a text alternative.
- Tests: the checks are the item.

## Maybe

### 33. Browser tests for the main flows

There are no browser tests. The Vitest page tests mock `fetch`, so a
mismatch between the API and the web types passes everywhere. A handful of
Playwright tests (sign in, add a lease, record a payment, see the balance)
against the Docker stack would catch it, at the price of a slow CI job.

### 34. Shared access for a partner or a manager

Everything belongs to one user (`owner_id`). Couples who own together, or a
part-time manager, share one password today. Real sharing means an account
that owns the data and members with roles (view only, can record payments,
everything), and every store query changes from `owner_id = $1` to a
membership check. That is the largest change on this list, which is why it
is a maybe.

### 35. Tenant portal

Tenants signing in to see their balance and receipts and to report a
repair would save the landlord messages. It needs a second kind of user,
invitations by email (item 20), and very careful scoping, built on the
membership model from item 34.

### 36. Online rent payments

Card or bank payments through a payment provider, recorded automatically
when they settle. It ties the app to an outside service and its fees, and
only makes sense once tenants can sign in (item 35).

### 37. Balances worked out in SQL

The rent roll and the dashboard load every charge and payment of every
lease (`LedgerEntries(ctx, ownerID, "")`) and apply payments in Go on each
request. That is quick for a few hundred leases. Beyond that the oldest
first allocation could be done with window functions, or balances could be
kept up to date on write. Only worth it with real numbers in hand, after
item 27.

### 38. Interest on deposits

Some places (several US states and cities, parts of Canada) require
interest to be paid on deposits held. A rate per property or per lease and
a yearly interest line in the deposit record would cover it. It is a niche
need and the rules differ widely.
