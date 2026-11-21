# Changelog

## Unreleased

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
