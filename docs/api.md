# API reference

The API is JSON over HTTP under `/api`. Everything except `/api/auth/register`,
`/api/auth/login` and `/healthz` needs an `Authorization: Bearer <token>`
header. See the README for the conventions on money, dates and errors.

Every record belongs to the signed in landlord. An ID that belongs to
someone else answers `404`, the same as one that does not exist.

## Account

| Method | Path                 | Notes                                                                  |
| ------ | -------------------- | ---------------------------------------------------------------------- |
| POST   | `/api/auth/register` | `{email, name, password}`, returns `{token, expires_at, user}`         |
| POST   | `/api/auth/login`    | `{email, password}`, same response                                     |
| GET    | `/api/auth/me`       | The signed in user                                                     |
| PUT    | `/api/auth/me`       | `{name, currency, deposit_return_days}`                                |
| PUT    | `/api/auth/password` | `{current_password, new_password}`; signs out other sessions and returns a new token |

`currency` is one of USD, CAD, EUR, GBP, AUD, NZD, CHF, SEK, PLN or RON. It
only changes how amounts are shown.

## Properties and units

| Method | Path                         | Notes                                       |
| ------ | ---------------------------- | ------------------------------------------- |
| GET    | `/api/properties`            | With unit counts and occupancy              |
| POST   | `/api/properties`            |                                             |
| GET    | `/api/properties/{id}`       |                                             |
| PUT    | `/api/properties/{id}`       |                                             |
| DELETE | `/api/properties/{id}`       | Refused while it has leases                 |
| GET    | `/api/properties/{id}/units` |                                             |
| POST   | `/api/properties/{id}/units` |                                             |
| GET    | `/api/units`                 | `?property_id=`                             |
| GET    | `/api/units/{id}`            | Current tenant, `vacant_since`, `next_lease_start` |
| PUT    | `/api/units/{id}`            |                                             |
| DELETE | `/api/units/{id}`            | Refused while it has leases                 |

## Tenants

| Method | Path                | Notes                                   |
| ------ | ------------------- | --------------------------------------- |
| GET    | `/api/tenants`      | `?q=` searches name, email and phone    |
| POST   | `/api/tenants`      |                                         |
| GET    | `/api/tenants/{id}` |                                         |
| PUT    | `/api/tenants/{id}` |                                         |
| DELETE | `/api/tenants/{id}` | Refused while they have leases          |

## Leases, rent and deposits

| Method | Path                              | Notes                                                        |
| ------ | --------------------------------- | ------------------------------------------------------------ |
| GET    | `/api/leases`                     | `?status=upcoming\|active\|ended\|terminated&property_id=&unit_id=&tenant_id=` |
| POST   | `/api/leases`                     | Rent due so far is charged straight away                     |
| GET    | `/api/leases/{id}`                | Includes `rent_changes`, `current_rent_cents` and renewal links |
| PUT    | `/api/leases/{id}`                | Charges already generated follow the new terms               |
| DELETE | `/api/leases/{id}`                | With its charges, payments and deposit records               |
| POST   | `/api/leases/{id}/terminate`      | `{date}`; the final month is prorated                        |
| POST   | `/api/leases/{id}/renew`          | `{end_date, rent_cents, notes}`; returns the new lease       |
| POST   | `/api/leases/{id}/rent-changes`   | `{effective_date, rent_cents, notes}`                        |
| DELETE | `/api/rent-changes/{id}`          |                                                              |
| GET    | `/api/leases/{id}/ledger`         | Charges with what is paid on each, payments and the balance  |
| POST   | `/api/leases/{id}/charges`        | `{kind: utility\|late_fee\|other, description, amount_cents, due_date}` |
| DELETE | `/api/charges/{id}`               |                                                              |
| POST   | `/api/leases/{id}/payments`       | `{amount_cents, paid_on, method, reference, notes}`          |
| GET    | `/api/leases/{id}/deposit`        | Agreed, received, returned, withheld and held amounts        |
| POST   | `/api/leases/{id}/deposit`        | `{kind: received\|returned\|withheld, amount_cents, occurred_on, notes}` |
| DELETE | `/api/deposit-transactions/{id}`  |                                                              |

## Payments and the rent roll

| Method | Path                         | Notes                                                       |
| ------ | ---------------------------- | ----------------------------------------------------------- |
| GET    | `/api/payments`              | `?lease_id=&property_id=&from=&to=&limit=&offset=`, newest first, at most 500 a page |
| GET    | `/api/payments/export.csv`   | Same filters, every match                                   |
| PUT    | `/api/payments/{id}`         | Correct a payment; the receipt number stays                 |
| DELETE | `/api/payments/{id}`         |                                                             |
| GET    | `/api/payments/{id}/receipt` | Landlord, tenant, property and the balance after it         |
| GET    | `/api/rent/balances`         | The rent roll; `?overdue=true` for late leases only         |
| POST   | `/api/rent/run`              | Generate due rent and late fees now                         |

## Maintenance, vendors and expenses

| Method | Path                            | Notes                                                    |
| ------ | ------------------------------- | -------------------------------------------------------- |
| GET    | `/api/maintenance`              | `?property_id=&status=active\|<status>&priority=&vendor_id=` |
| POST   | `/api/maintenance`              |                                                          |
| GET    | `/api/maintenance/{id}`         | Includes `next_statuses`                                 |
| PUT    | `/api/maintenance/{id}`         |                                                          |
| POST   | `/api/maintenance/{id}/status`  | `{status}`, following the workflow                       |
| DELETE | `/api/maintenance/{id}`         |                                                          |
| GET    | `/api/vendors`                  | `?trade=`, with open jobs and total spent                |
| POST   | `/api/vendors`                  |                                                          |
| GET    | `/api/vendors/{id}`             |                                                          |
| PUT    | `/api/vendors/{id}`             |                                                          |
| DELETE | `/api/vendors/{id}`             | Jobs and expenses keep the name                          |
| GET    | `/api/expenses`                 | `?property_id=&category=&vendor_id=&from=&to=`, with totals by category |
| GET    | `/api/expenses/export.csv`      | Same filters                                             |
| POST   | `/api/expenses`                 |                                                          |
| GET    | `/api/expenses/{id}`            |                                                          |
| PUT    | `/api/expenses/{id}`            |                                                          |
| DELETE | `/api/expenses/{id}`            |                                                          |
| GET    | `/api/recurring-expenses`       | `?property_id=`, with `next_on`                          |
| POST   | `/api/recurring-expenses`       | `{..., day_of_month, every_months: 1\|3\|6\|12, start_date, end_date}` |
| GET    | `/api/recurring-expenses/{id}`  |                                                          |
| PUT    | `/api/recurring-expenses/{id}`  | Later occurrences use the new terms                      |
| DELETE | `/api/recurring-expenses/{id}`  | Expenses already created are kept                        |

## Documents

| Method | Path                           | Notes                                                       |
| ------ | ------------------------------ | ----------------------------------------------------------- |
| GET    | `/api/documents`               | Exactly one of `?tenant_id=`, `?lease_id=` or `?property_id=` |
| POST   | `/api/documents`               | Multipart form: one of the IDs above, `file` and `notes`; up to 10 MB |
| GET    | `/api/documents/{id}/download` | Always sent as an attachment                                |
| DELETE | `/api/documents/{id}`          |                                                             |

PDFs, images (JPEG, PNG, GIF, WebP, HEIC), text, CSV and Word, Excel or
OpenDocument files are accepted.

## Overview and reports

| Method | Path                           | Notes                                                   |
| ------ | ------------------------------ | ------------------------------------------------------- |
| GET    | `/api/dashboard`               | Occupancy, this month, overdue rent, deposits held, leases ending, maintenance and 12 months of totals |
| GET    | `/api/reminders`               | What needs doing soon, most pressing first              |
| GET    | `/api/reports/profit-loss`     | `?from=&to=` (default: the year to date), per property  |
| GET    | `/api/reports/profit-loss.csv` | The same as a spreadsheet                               |
| GET    | `/api/activity`                | `?lease_id=&entity_type=&entity_id=&before=&limit=`, newest first |
| GET    | `/api/search`                  | `?q=` with at least two characters                      |
| GET    | `/healthz`                     | `200` when the database answers                         |
