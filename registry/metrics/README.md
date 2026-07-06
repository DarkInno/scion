# Metrics Module

Prometheus HTTP metrics for `net/http` services.

## What's Included

- Isolated Prometheus registry
- HTTP request counter
- Request duration histogram
- In-flight request gauge
- `/metrics` handler
- Route label cardinality limits
- CRLF, null byte, raw URL, and long-label protection

## Quick Copy

This module uses Prometheus and should be copied in standalone mode:

```bash
scion add metrics --standalone --to internal/metrics
```

## Usage

```go
m, err := metrics.New()
if err != nil {
    return err
}
_ = m.RegisterDefaults()

http.Handle("/metrics", m.Handler())
http.Handle("/users/", m.Middleware("/users/{id}")(usersHandler))
```

Always pass a route template such as `/users/{id}`. Do not pass `r.URL.Path`
or a full URL because that can create unbounded label cardinality.

## File Reference

| File | Purpose |
|------|---------|
| `config.go` | Options, defaults, env loading |
| `metrics.go` | Registry and collectors |
| `middleware.go` | HTTP instrumentation middleware |
| `handler.go` | Prometheus scrape handler |
| `pentest_test.go` | Attack-scenario tests |

## Tests

```bash
cd registry/metrics/src/go
go test -v ./...
```
