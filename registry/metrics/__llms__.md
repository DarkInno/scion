# metrics module

Prometheus-backed Go HTTP metrics module. Copy standalone with `go.mod`/`go.sum` because it depends on `github.com/prometheus/client_golang`. Use `metrics.New()`, optional `RegisterDefaults()`, `m.Handler()` for `/metrics`, and `m.Middleware("/route/{id}")`. Never pass raw URL paths as labels. Enforces route/method length checks, CRLF/null-byte rejection, and max route-cardinality overflow.
