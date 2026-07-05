# Scion

> Graft backend patterns into your project. Copy-paste, not install.

[English](README.md) | [中文](README_zh.md)

Scion is a copy-paste code library for Go backend development. Instead of installing a framework or pulling a dependency, you copy pre-built, production-ready modules into your project and own every line of code.

## Why Copy-Paste?

Backend modules (auth, CRUD, file upload, rate limiting) share 80% of their skeleton across projects, but the remaining 20% differs in ways that make npm/go packages awkward:

- You need to customize business logic deep inside the module
- You want to own the code, not be locked to upstream versions
- Your AI coding assistant works better with code it can read and modify directly
- No dependency hell — zero external dependencies, Go standard library only

## Quick Start

```bash
# 1. Copy a module into your project
cp -r registry/auth/src/go/* yourproject/internal/auth/

# 2. Adapt the configuration
#    Edit config.go: set JWT secret, database URL, etc.

# 3. Implement the store interface
#    type UserStore interface { ... }  // your DB layer

# 4. Wire up routes
#    See registry/auth/examples/gin/main.go
```

## Available Modules

| Module | Description | Security Features |
|--------|-------------|-------------------|
| [auth](registry/auth/) | JWT email/password auth + bcrypt | Rate limiting, user enumeration prevention, JTI, aud/iss validation |
| [crud](registry/crud/) | Generic CRUD with pagination | Sort/filter whitelist, SQL injection prevention, pagination ceiling |
| [middleware](registry/middleware/) | Recovery, CORS, logging, timeout, etc. | CRLF injection prevention, trusted proxy, body size limit |
| [rbac](registry/rbac/) | Role-based access control | Wildcard permissions, cycle detection, hierarchy inheritance |
| [ratelimit](registry/ratelimit/) | Fixed window / sliding window / token bucket | Memory exhaustion protection, LRU eviction, key length limit |
| [validation](registry/validation/) | Chainable request validation builder | Regex DoS prevention (RE2), null byte/CRLF rejection, panic recovery |
| [file-upload](registry/file-upload/) | Secure file upload handler | Magic bytes validation, path traversal prevention, size limit, rate limiting |
| [health](registry/health/) | Liveness/readiness probes | SSRF protection (private IP rejection), CRLF injection prevention |
| [cache](registry/cache/) | Generic TTL + LRU cache | Background cleanup, goroutine leak prevention, max entries limit |
| [pagination](registry/pagination/) | Offset/limit + cursor pagination | Cursor base64 validation, negative offset clamp, max limit enforcement |
| [mail](registry/mail/) | SMTP email with templates | Header injection prevention, XSS escaping, attachment sanitization, async queue |

## Project Structure

```
scion/
├── registry/
│   ├── index.json              # Machine-readable module index
│   ├── auth/                   # Authentication module
│   │   ├── __llms__.md         # AI-readable summary (~150 tokens)
│   │   ├── README.md           # Human-readable adaptation guide
│   │   ├── src/go/             # Go source code
│   │   └── examples/gin/       # Minimal runnable example
│   ├── crud/                   # CRUD operations module
│   ├── middleware/             # HTTP middleware collection
│   ├── rbac/                   # Role-based access control
│   ├── ratelimit/              # Rate limiting algorithms
│   ├── validation/             # Request validation builder
│   ├── file-upload/            # File upload handler
│   ├── health/                 # Health check probes
│   ├── cache/                  # In-memory cache
│   ├── pagination/             # Pagination utilities
│   └── mail/                   # Email sender
├── docs/
│   └── getting-started.md      # How to use Scion
├── AGENTS.md                   # AI coding agent instructions
├── CONTRIBUTING.md             # How to contribute
├── LICENSE                     # MIT
└── llms.txt                    # LLM-friendly project summary
```

## Design Principles

1. **Code ownership** — every line is yours after copying. No upstream lock-in.
2. **Self-contained** — each module works independently, zero external dependencies.
3. **Framework-agnostic** — uses Go standard `net/http`, adaptable to Gin/Echo/etc.
4. **Security-first** — input validation, rate limiting, injection prevention built in.
5. **AI-friendly** — `__llms__.md` files let AI assistants understand modules in ~200 tokens.
6. **Tested** — every module includes functional tests and penetration test cases.

## Development

```bash
# Clone the repository
git clone https://github.com/your-org/scion.git
cd scion

# Run tests for a specific module
cd registry/auth/src/go && go test -v ./...

# Run tests for all modules
# (PowerShell)
$modules = @('middleware','auth','crud','rbac','ratelimit','validation','file-upload','health','cache','pagination','mail')
foreach ($m in $modules) { Push-Location "registry/$m/src/go"; go test ./...; Pop-Location }

# Format code
cd registry/auth/src/go && gofmt -w .
```

## Contributing

We welcome contributions! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on adding new modules.

## License

[MIT](LICENSE)
