# Health Module

Liveness, readiness, and health probes for `net/http` services.

## What's Included

- Health checker registry
- Liveness, readiness, and combined health HTTP handlers
- Custom checks
- HTTP checks with SSRF protection
- TCP checks
- CRLF, null-byte, name, and URL length validation

## Quick Copy

```bash
cp -r registry/health/src/go/*.go yourproject/internal/health/
```

Or with the Scion CLI:

```bash
scion add health --to internal/health
```

## Usage

```go
checker := health.New()
check, _ := health.NewCustomCheck("database", func(ctx context.Context) error {
	return db.PingContext(ctx)
})
_ = checker.AddCheck(check)

h := health.NewHealthHandler(checker)
http.HandleFunc("/live", h.Liveness)
http.HandleFunc("/ready", h.Readiness)
```

## File Reference

| File | Purpose |
|------|---------|
| `checker.go` | Check registration and execution |
| `checks.go` | HTTP, TCP, and custom check implementations |
| `handler.go` | JSON HTTP handlers |
| `pentest_test.go` | SSRF and injection tests |

## Tests

```bash
cd registry/health/src/go
go test -v ./...
```
