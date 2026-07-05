# Auth Module

JWT-based email/password authentication with bcrypt.

## What's Included

- User registration and login
- Password hashing (bcrypt, cost 12 by default, configurable 10-15)
- JWT access token generation and validation (HS256, JTI, aud, iss, nbf)
- Middleware for protecting routes
- Rate limiting hooks (email + IP) with included memory implementation
- User enumeration prevention

## Quick Copy

```bash
cp -r registry/auth/src/go/* yourproject/internal/auth/
```

## Adaptation Guide

### 1. Database Layer

Implement the `auth.UserStore` interface:

```go
type UserStore interface {
    Create(ctx context.Context, user *User) error
    GetByEmail(ctx context.Context, email string) (*User, error)
    GetByID(ctx context.Context, id string) (*User, error)
}
```

### 2. Configuration

Set environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes | - | Min 32 chars, max 512 |
| `DB_URL` | Yes | - | Database connection string |
| `TOKEN_EXPIRY` | No | 3600 | Token expiry in seconds (max 604800) |
| `JWT_ISSUER` | No | Scion-auth | Issuer for aud/iss validation |
| `BCRYPT_COST` | No | 12 | Bcrypt cost (10-15) |

### 3. Rate Limiting

```go
rl := auth.NewMemoryRateLimiter(10, 15*time.Minute)
handler := auth.NewHandler(store, cfg).WithRateLimiter(rl)
```

### 4. Routes

Default prefix: `/api/v1/auth`

Change with `auth.RoutePrefix` before registering routes.

## File Reference

| File | Purpose |
|------|---------|
| `config.go` | Env var loading and validation |
| `models.go` | User struct, request/response types |
| `password.go` | Bcrypt hash and verify |
| `jwt.go` | Token generation and parsing |
| `handlers.go` | HTTP handlers (register, login, me) |
| `middleware.go` | JWT Bearer validation middleware |
| `routes.go` | Route registration |
| `ratelimiter.go` | In-memory sliding-window rate limiter |

## Security Checklist

- [ ] `JWT_SECRET` is at least 32 random characters
- [ ] `BCRYPT_COST` is between 10 and 15
- [ ] Email addresses are normalized before storage/lookup
- [ ] Rate limiting is configured (email + IP)
- [ ] `JWT_ISSUER` matches across all services

## Tests

```bash
cd registry/auth/src/go
go test -v ./...
```

## Example

See `registry/auth/examples/gin/` for a minimal runnable example.
