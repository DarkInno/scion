# Modules Overview

Scion provides 12 production-ready, copy-paste Go modules. Each module is self-contained. Modules are standard-library only by default; declared security exceptions are marked in the registry.

## Available Modules

| Module | Description | Security Features |
|--------|-------------|-------------------|
| [Auth](/modules/auth) | JWT email/password auth + bcrypt | Rate limiting, user enumeration prevention, JTI |
| [CRUD](/modules/crud) | Generic CRUD with pagination | Sort/filter whitelist, SQL injection prevention |
| [Database](/modules/database) | `database/sql` setup + transactions | DSN-safe errors, whitelisted SQL fragments |
| [Middleware](/modules/middleware) | Recovery, CORS, logging, timeout | CRLF injection prevention, body size limit |
| [RBAC](/modules/rbac) | Role-based access control | Wildcard permissions, cycle detection |
| [Rate Limit](/modules/ratelimit) | Fixed/sliding window, token bucket | Memory exhaustion protection, LRU eviction |
| [Validation](/modules/validation) | Chainable request validation | Regex DoS prevention, null byte rejection |
| [File Upload](/modules/file-upload) | Secure file upload handler | Magic bytes validation, path traversal prevention |
| [Health](/modules/health) | Liveness/readiness probes | SSRF protection, CRLF injection prevention |
| [Cache](/modules/cache) | TTL + LRU in-memory cache | Background cleanup, max entries limit |
| [Pagination](/modules/pagination) | Offset/cursor pagination | Cursor base64 validation, max limit enforcement |
| [Mail](/modules/mail) | SMTP email with templates | Header injection prevention, XSS escaping |

## Quick Copy

```bash
# Copy a module into your project
cp -r registry/<module>/src/go/* yourproject/internal/<module>/
```

## Module Structure

Each module follows this structure:

```
registry/<module>/
├── src/go/
│   ├── go.mod              # module <name>, go 1.22
│   ├── config.go           # Options struct, Defaults(), FromEnv()
│   ├── handler.go          # HTTP handlers
│   ├── <core>.go           # Core logic
│   ├── <core>_test.go      # Functional tests
│   └── pentest_test.go     # Penetration test cases
├── README.md               # Human-readable adaptation guide
└── __llms__.md             # AI-readable summary (~150 tokens)
```

## Testing

Every module includes functional tests and penetration test cases:

```bash
cd registry/<module>/src/go
go test -v ./...
```

## Dependencies

Modules use only the Go standard library by default. Declared exceptions, such as auth, copy their own `go.mod` in standalone mode.
