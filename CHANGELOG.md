# Changelog

## Unreleased

- Password reset by email: "Forgot your password?" sends a one-use link
  valid for 30 minutes, and setting the new password signs out every
  device. Needs `SMTP_HOST`, `MAIL_FROM` and `APP_URL`; new endpoints
  `POST /api/auth/forgot` and `POST /api/auth/reset`.

- Tenant statements: a printable page per lease with the opening balance,
  every charge and payment in a period with the running balance, and the
  closing balance. `GET /api/leases/{id}/statement?from=&to=`.

- Scheduled backups: a `backup` service in Docker Compose dumps the
  database into `./backups` every `BACKUP_INTERVAL` (a day) and keeps the
  newest `BACKUP_KEEP` (14). `make backup` and `make restore FILE=...`.
- `docs/backups.md` on restoring, checking backups, copies off the machine
  and upgrading.

- A time zone setting. Rent, late fees, recurring expenses, reminders and
  "this month" follow the landlord's date instead of the UTC date. Accounts
  start in UTC, so nothing moves until a zone is picked. The signed in user
  carries `today` in their zone, and the web forms use the account's zone
  for today's date.

- The web app is served with a content security policy, `nosniff`,
  `X-Frame-Options: DENY`, a referrer policy and a permissions policy.
- Every API response is `no-store` and carries a policy that allows nothing
  to run or frame it.

- Sign-in is rate limited by address, and by email and address together, so
  guessing from one address cannot lock the owner out. Sign-up is limited
  by address. Over a limit the API answers `429` with `Retry-After`.
- `ALLOW_REGISTRATION=false` closes sign-up once the first account exists.
  The sign-in page hides the link and the sign-up page says it is closed.
- `TRUSTED_PROXIES` lists the proxies whose `X-Forwarded-For` header is
  believed. Docker Compose pins its network (`SUBNET`, `WEB_IP`) and trusts
  only its nginx.
- Docker Compose publishes the API port on `127.0.0.1` only.
- New `GET /api/auth/config` endpoint.
