# Metrics Module

Prometheus HTTP metrics for `net/http` services.

## Features

- Isolated Prometheus registry
- Request counter, duration histogram, and in-flight gauge
- `/metrics` scrape handler
- `func(http.Handler) http.Handler` middleware
- Optional Go runtime/process collectors

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

Pass route templates, not raw URLs.

## Security

- Rejects CRLF and null bytes in labels
- Normalizes raw URLs and overlong labels
- Caps route-label cardinality and sends overflow to `route="overflow"`

## Copy

`metrics` uses Prometheus, so copy it in standalone mode:

```bash
scion add metrics --standalone --to internal/metrics
```
