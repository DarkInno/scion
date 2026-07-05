# Getting Started

## 1. Choose a Module

Browse the [Modules](/modules/) section or the `registry/` directory to find what you need.

## 2. Copy It

```bash
# Example: copy the auth module
cp -r registry/auth/src/go/* yourproject/internal/auth/
```

## 3. Adapt It

Each module has a `README.md` with an adaptation checklist:

1. **Database layer** — implement the store interface
2. **Configuration** — set environment variables
3. **Routes** — adjust prefix if needed

## 4. Run It

```bash
# Run tests
cd yourproject/internal/auth
go test -v ./...
```

## Available Modules

| Module | Description |
|--------|-------------|
| [Auth](/modules/auth) | JWT authentication with bcrypt |
| [CRUD](/modules/crud) | Generic CRUD with pagination |
| [Middleware](/modules/middleware) | Recovery, CORS, logging, timeout |
| [RBAC](/modules/rbac) | Role-based access control |
| [Rate Limit](/modules/ratelimit) | Fixed/sliding window, token bucket |
| [Validation](/modules/validation) | Chainable request validation |
| [File Upload](/modules/file-upload) | Secure file upload handler |
| [Health](/modules/health) | Liveness/readiness probes |
| [Cache](/modules/cache) | TTL + LRU in-memory cache |
| [Pagination](/modules/pagination) | Offset/cursor pagination |
| [Mail](/modules/mail) | SMTP email with templates |

## Project Structure

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

## Next Steps

- Read [Why Copy-Paste?](/guide/why-copy-paste) to understand the philosophy
- Check [Security Design](/guide/security) for security best practices
- Browse [Modules](/modules/) to find what you need
