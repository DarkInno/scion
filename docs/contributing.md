# Contributing

We welcome contributions! Here's how to add new modules to Scion.

## Adding a New Module

### 1. Create Module Structure

```
registry/<module-name>/
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

### 2. Follow Security Requirements

Every module MUST implement:

- **CRLF injection prevention** — reject `\r\n` in all user inputs
- **Null byte rejection** — reject `\x00` in all string inputs
- **Length limits** — all user-supplied strings have max length checks
- **Memory exhaustion protection** — maps/slices with unbounded growth must have limits
- **No X-Forwarded-For trust** — `ClientIP()` returns `r.RemoteAddr` only
- **Path traversal prevention** — use `filepath.Base()` + reject `..`
- **Parameterized queries** — never concatenate user input into SQL
- **Panic recovery** — all HTTP handlers must recover from panics

### 3. Write Tests

Every source file needs a corresponding `_test.go`:

```bash
cd registry/<module>/src/go
go test -v ./...
```

### 4. Add Documentation

- `README.md` — human-readable adaptation guide
- `__llms__.md` — AI-readable summary (~150 tokens)

### 5. Update Registry Index

Add your module to `registry/index.json`.

## Code Standards

- Go 1.22+ with generics
- Zero external dependencies — standard library only
- `gofmt` formatting
- `go vet` must pass
- Middleware signature: `func(http.Handler) http.Handler`
- Use `log/slog` for logging

## Testing

```bash
# Test one module
cd registry/<module>/src/go && go test -v ./...

# Test all modules
$modules = @('middleware','auth','crud','rbac','ratelimit','validation','file-upload','health','cache','pagination','mail')
foreach ($m in $modules) { Push-Location "registry/$m/src/go"; go test ./...; Pop-Location }
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Add your module following the guidelines above
4. Run all tests
5. Submit a pull request

## Questions?

Open an issue on GitHub or join the discussion.
