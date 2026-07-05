# Middleware Module

Framework-agnostic HTTP middleware for standard `net/http` handlers.

## What's Included

- Middleware chaining
- Panic recovery
- CORS
- Request logging with `log/slog`
- Request ID
- Timeout
- Body size limit
- Trusted proxy helper
- Debug and trace helpers

## Quick Copy

```bash
cp -r registry/middleware/src/go/*.go yourproject/internal/middleware/
```

Or with the Scion CLI:

```bash
scion add middleware --to internal/middleware
```

## Usage

```go
handler := middleware.Chain(
	middleware.Recovery(),
	middleware.RequestID(),
	middleware.Logging(),
	middleware.BodyLimit(10<<20),
)(mux)
```

## File Reference

| File | Purpose |
|------|---------|
| `chain.go` | Middleware composition |
| `config.go` | Shared security defaults |
| `recovery.go` | Panic recovery |
| `cors.go` | CORS handling |
| `logging.go` | Structured request logging |
| `requestid.go` | Request ID generation and validation |
| `timeout.go` | Request timeout middleware |
| `bodylimit.go` | Request body limits |
| `proxy.go` | Trusted proxy support |
| `trace.go` | Trace ID helpers |
| `debug.go` | Debug route guard |
| `context.go` | Context helper keys |
| `rw.go` | Response writer wrapper |

## Tests

```bash
cd registry/middleware/src/go
go test -v ./...
```
