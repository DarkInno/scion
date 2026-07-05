# auth module

JWT-based email/password authentication with optional OAuth2 (Google, GitHub).

## Languages
- Go 1.22+ (src/go/)
- Python 3.12+ (src/python/)

## Config (Go)
- JWT_SECRET — signing key, min 32 chars, max 512
- DB_URL — database connection
- TOKEN_EXPIRY — seconds, default 3600, max 604800 (7 days)
- JWT_ISSUER — token issuer, default "Scion-auth"
- BCRYPT_COST — 10-15, default 12. Cost > 15 risks DoS via CPU exhaustion
- OAUTH_GOOGLE_CLIENT_ID — optional
- OAUTH_GITHUB_CLIENT_ID — optional

## Security Features (Go)
- bcrypt with 72-byte input limit (prevents silent truncation)
- JWT JTI claim for token revocation (use JTI as blocklist key, not raw JWT)
- JWT aud/iss validation prevents cross-service token reuse
- alg=none attack prevention (HMAC-only, reject none/RS/ES)
- Input validation: email format, password 8-72 chars
- NormalizeEmail for case-insensitive matching
- Rate limiting by email AND IP (prevents cycling emails)
- User enumeration prevention (generic error messages)
- Request body limited to 1MB
- SecureCompare for non-bcrypt secrets

## Adapt (Go)
- Implement `UserStore` interface with your DB layer (GORM, sqlx, pgx)
- Call `NormalizeEmail()` before storing/looking up emails
- Copy `src/go/` into your project, run `go mod tidy`
- Set env vars
- Optional: provide `RateLimiter` via `handler.WithRateLimiter(rl)`
- Optional: customize `RoutePrefix` before `RegisterRoutes()`

## Deps (Go)
golang-jwt/jwt/v5, golang.org/x/crypto/bcrypt, joho/godotenv

## Deps (Python)
SQLAlchemy 2.0, python-jose, passlib
