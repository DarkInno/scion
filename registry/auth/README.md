# Auth Module

JWT-based email/password authentication with optional OAuth2 integration.

## What's Included

- User registration and login
- Password hashing (bcrypt, cost 12 by default, configurable 10-15)
- JWT access token generation and validation (HS256, JTI, aud, iss, nbf)
- Optional OAuth2 login (Google, GitHub)
- Middleware for protecting routes
- Rate limiting hooks (email + IP) with included memory implementation
- User enumeration prevention

## Quick Copy

### Go

```bash
cp -r registry/auth/src/go/* src/auth/
```

### Python

```bash
cp -r registry/auth/src/python/* src/auth/
cp -r registry/auth/examples/fastapi/* src/auth/
```

## Adaptation Guide (Go)

1. **Database layer** — implement `auth.UserStore` interface
2. **Email normalization** — call `auth.NormalizeEmail()` before storing/looking up
3. **Configuration** — set environment variables:
   - `JWT_SECRET` — required, min 32 chars, max 512
   - `DB_URL` — required
   - `TOKEN_EXPIRY` — default 3600 seconds, max 604800 (7 days)
   - `JWT_ISSUER` — default "Scion-auth", used for aud/iss validation
   - `BCRYPT_COST` — 10-15, default 12. Above 15 risks DoS via CPU exhaustion
   - `OAUTH_GOOGLE_CLIENT_ID` — optional
   - `OAUTH_GITHUB_CLIENT_ID` — optional
4. **Rate limiting** (optional but recommended):
   ```go
   rl := auth.NewMemoryRateLimiter(10, 15*time.Minute)
   handler := auth.NewHandler(store, cfg).WithRateLimiter(rl)
   ```
   For distributed deployments, replace with a Redis-backed implementation.
5. **Routes prefix** — default `/api/v1/auth`. Change `auth.RoutePrefix` before registering routes, or use `auth.ValidateRoutePrefix()` to validate a custom prefix

## File Reference (Go)

| File | Purpose |
|------|---------|
| `config.go` | Env var loading and validation |
| `models.go` | User struct, request/response types, input validation, ClientIP helper |
| `password.go` | Bcrypt hash and verify, 72-byte limit, SecureCompare |
| `jwt.go` | Token generation and parsing (HS256, JTI, aud, iss, nbf) |
| `handlers.go` | HTTP handlers (register, login, me), body size limit, rate limiting |
| `middleware.go` | JWT Bearer validation middleware |
| `routes.go` | Route registration, RoutePrefix, ValidateRoutePrefix |
| `ratelimiter.go` | In-memory sliding-window rate limiter |

## Tests

Every file has corresponding `*_test.go` coverage:

```bash
cd registry/auth/src/go
go test -v ./...
```

Test coverage includes:
- Config validation (secret length, expiry capping, bcrypt bounds)
- JWT generation and parsing (wrong secret, expired, invalid issuer/audience, non-HMAC rejection)
- Password hashing (72-byte limit, cost bounds, different salts)
- Rate limiting (sliding window, concurrent access, per-key isolation)
- HTTP handlers (register, login, me) with mock store
- Middleware (valid token, missing header, invalid format, expired token)
- Routes and input validation

## Security Checklist

When adapting this module, ensure:
- [ ] `JWT_SECRET` is at least 32 random characters
- [ ] `BCRYPT_COST` is between 10 and 15
- [ ] Email addresses are normalized with `NormalizeEmail()` before storage/lookup
- [ ] Rate limiting is configured (email + IP)
- [ ] `JWT_ISSUER` matches across all services sharing the same JWT secret

## Example Usage

See `examples/gin/` for a template Go project.
