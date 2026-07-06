# AGENTS.md

> Instructions for AI coding agents (Codex, Claude, Cursor, etc.) working on this project.

[English](AGENTS.md) | [中文](AGENTS_zh.md)

## Project Description

Scion is a copy-paste code library for Go backend development. It contains 15 self-contained modules in `registry/` — each is a standalone Go package. Modules are standard-library only by default; security-sensitive or observability modules may be declared as `stdlibOnly:false` in `registry/index.json` and copied in standalone mode. Modules are meant to be copied into a user's project and adapted, not imported as a dependency.

## Coding Standards

- Go 1.22+ with generics
- Standard library only by default; declared `stdlibOnly:false` modules may use mature security or observability libraries
- `gofmt` formatting is mandatory
- `go vet` must pass with zero warnings
- Middleware signature: `func(http.Handler) http.Handler`
- All `json.NewEncoder(w).Encode()` errors must be explicitly ignored with `_ =`
- `defer r.Body.Close()` goes after `io.ReadAll`, not before (Go http.Server closes body automatically)
- Use `log/slog` for logging — never `fmt.Println` or `log.Printf`

## Module Conventions

Each module lives in `registry/<module-name>/src/go/` with this structure:

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

## Security Requirements (Non-Negotiable)

Every module MUST implement these:

- **CRLF injection prevention** — reject `\r\n` in all user inputs (headers, URLs, names)
- **Null byte rejection** — reject `\x00` in all string inputs
- **Length limits** — all user-supplied strings have max length checks
- **Memory exhaustion protection** — maps/slices with unbounded growth MUST have `maxBuckets` or `maxEntries` limits + LRU eviction
- **No X-Forwarded-For trust** — `ClientIP()` must return `r.RemoteAddr` only; XFF is client-controlled and spoofable
- **Path traversal prevention** — use `filepath.Base()` + reject `..` in all file operations
- **Parameterized queries** — never concatenate user input into SQL
- **Panic recovery** — all HTTP handlers must recover from panics

## Testing Requirements

- Every source file has a corresponding `_test.go`
- Every module has a `pentest_test.go` with attack-scenario test cases
- Run tests before any commit:

```bash
cd registry/<module>/src/go && go test -v -count=1 ./...
```

- Run all module tests:

```bash
# PowerShell
$modules = @('middleware','auth','crud','database','rbac','ratelimit','validation','file-upload','health','cache','pagination','mail','migrations','metrics','problem')
foreach ($m in $modules) { Push-Location "registry/$m/src/go"; go test ./...; Pop-Location }
```

## Key Constraints

- Do NOT add external dependencies to any module's `go.mod` unless the module is explicitly marked `stdlibOnly:false` in `registry/index.json` and the dependency is justified for security, correctness, or observability
- Do NOT use `panic` in HTTP handlers — return errors
- Do NOT trust client headers (`Content-Type`, `X-Forwarded-For`, `X-Real-Ip`)
- Do NOT use `strings.Split` for header parsing — use `strings.SplitN` with a limit
- Do NOT modify `go.mod` files to add dependencies without explicit user approval
- Configuration must live in environment variables or a `config.go` with `FromEnv()`
- All modules must be framework-agnostic (use `net/http`, not Gin/Echo directly)

## AI Prompt Template

When you want an AI assistant to work on this project, copy the prompt below:

---

```
You are working on Scion, a copy-paste code library for Go backend development.

Project location: <path-to-scion>

Architecture:
- 15 modules in registry/ — each is a standalone Go package
- Module path pattern: registry/<module>/src/go/
- Go 1.22+, standard library by default, gofmt mandatory

Security rules (non-negotiable):
1. Reject CRLF (\r\n) and null bytes (\x00) in all user inputs
2. All strings have max length checks (128-1024 chars depending on context)
3. Maps with unbounded growth MUST have maxBuckets/maxEntries + LRU eviction
4. ClientIP() returns r.RemoteAddr only — never trust X-Forwarded-For
5. Use filepath.Base() + reject ".." for all file path operations
6. All HTTP handlers must recover from panics
7. All json.NewEncoder(w).Encode() errors must use `_ =` to ignore

Test requirements:
- Every module has pentest_test.go with attack scenarios
- Run: cd registry/<module>/src/go && go test -v -count=1 ./...

Task: <describe your task here>
```

---

## Common Commands

| Action | Command |
|--------|---------|
| Test one module | `cd registry/auth/src/go && go test -v ./...` |
| Test all modules | See PowerShell snippet above |
| Format code | `cd registry/<module>/src/go && gofmt -w .` |
| Vet code | `cd registry/<module>/src/go && go vet ./...` |
| Coverage | `cd registry/<module>/src/go && go test -cover ./...` |

## Directory Structure

```
scion/
├── registry/                 # 15 copy-paste modules
│   ├── auth/                 # JWT auth + bcrypt
│   ├── crud/                 # Generic CRUD + pagination
│   ├── database/             # database/sql helpers
│   ├── middleware/           # 9 HTTP middlewares
│   ├── rbac/                 # Role-based access control
│   ├── ratelimit/            # 3 rate limiting algorithms
│   ├── validation/           # Chainable validation builder
│   ├── file-upload/          # Secure file upload
│   ├── health/               # Liveness/readiness probes
│   ├── cache/                # TTL + LRU cache
│   ├── pagination/           # Offset + cursor pagination
│   ├── mail/                 # SMTP email sender
│   ├── migrations/           # SQL migration runner
│   ├── metrics/              # Prometheus HTTP metrics
│   └── problem/              # RFC 9457 problem responses
├── docs/                     # Human-readable docs
├── AGENTS.md                 # This file (English)
├── AGENTS_zh.md              # This file (Chinese)
├── CONTRIBUTING.md           # Contribution guide
└── LICENSE                   # MIT
```
