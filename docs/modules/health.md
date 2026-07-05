# Health Module

Liveness and readiness probes with SSRF protection.

## What's Included

- Liveness probe (is the service alive?)
- Readiness probe (is the service ready to accept traffic?)
- Custom health checks
- SSRF protection (private IP rejection)

## Quick Copy

```bash
cp -r registry/health/src/go/* yourproject/internal/health/
```

## Usage

### Basic Setup

```go
checker := health.NewChecker()

// Add readiness checks
checker.AddCheck("database", func(ctx context.Context) error {
    return db.PingContext(ctx)
})

checker.AddCheck("redis", func(ctx context.Context) error {
    return redis.Ping(ctx).Err()
})

// Register handlers
http.Handle("/healthz", checker.LivenessHandler())
http.Handle("/readyz", checker.ReadinessHandler())
```

### Custom Checks

```go
checker.AddCheck("external-api", func(ctx context.Context) error {
    resp, err := http.Get("https://api.example.com/health")
    if err != nil {
        return err
    }
    if resp.StatusCode != 200 {
        return fmt.Errorf("status: %d", resp.StatusCode)
    }
    return nil
})
```

## Endpoints

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `/healthz` | Liveness probe | 200 OK if alive |
| `/readyz` | Readiness probe | 200 OK if all checks pass |

## Configuration

```go
checker := health.NewChecker(health.Config{
    Timeout: 5 * time.Second,
})
```

## File Reference

| File | Purpose |
|------|---------|
| `checker.go` | Health check manager |
| `handler.go` | HTTP handlers |
| `checks.go` | Built-in checks |

## Security Features

- SSRF protection (private IP rejection in HTTP checks)
- CRLF injection prevention
- Timeout on all checks

## Tests

```bash
cd registry/health/src/go
go test -v ./...
```
