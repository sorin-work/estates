# Changelog

## Unreleased

- Sign-in is rate limited by address and by email, and sign-up by address.
  Over a limit the API answers `429` with `Retry-After`.
- `ALLOW_REGISTRATION=false` closes sign-up once the first account exists.
  The sign-in page hides the link and the sign-up page says it is closed.
- `TRUSTED_PROXIES` lists the proxies whose `X-Forwarded-For` header is
  believed. Docker Compose sets it for its nginx.
- New `GET /api/auth/config` endpoint.
