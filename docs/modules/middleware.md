# Middleware Module

HTTP middleware collection for Go web applications.

## What's Included

- **Recovery** — panic recovery with logging
- **CORS** — Cross-Origin Resource Sharing
- **Logging** — request/response logging
- **Timeout** — request timeout
- **Request ID** — unique request identifier
- **Body Limit** — request body size limit
- **Proxy** — trusted proxy handling
- **Debug** — debug mode detection

## Quick Copy

```bash
cp -r registry/middleware/src/go/* yourproject/internal/middleware/
```

## Usage

### Chain Middlewares

```go
handler := middleware.Chain(
    middleware.Recovery(),
    middleware.CORS(middleware.CORSConfig{
        AllowOrigins: []string{"https://example.com"},
        AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    }),
    middleware.Logging(),
    middleware.Timeout(30 * time.Second),
    middleware.BodyLimit(10 << 20), // 10 MB
)(yourHandler)
```

### Individual Middleware

```go
// Recovery
http.Handle("/api", middleware.Recovery()(handler))

// CORS
http.Handle("/api", middleware.CORS(config)(handler))

// Logging
http.Handle("/api", middleware.Logging()(handler))
```

## Configuration

### CORS

```go
config := middleware.CORSConfig{
    AllowOrigins: []string{"https://example.com"},
    AllowMethods: []string{"GET", "POST"},
    AllowHeaders: []string{"Content-Type", "Authorization"},
    MaxAge: 86400,
}
```

### Timeout

```go
handler := middleware.Timeout(30 * time.Second)(handler)
```

## File Reference

| File | Purpose |
|------|---------|
| `recovery.go` | Panic recovery middleware |
| `cors.go` | CORS middleware |
| `logging.go` | Request logging |
| `timeout.go` | Request timeout |
| `requestid.go` | Request ID generation |
| `bodylimit.go` | Body size limit |
| `chain.go` | Middleware chaining |
| `config.go` | Configuration |

## Security Features

- CRLF injection prevention in headers
- Trusted proxy validation
- Body size limit prevents large payload attacks

## Tests

```bash
cd registry/middleware/src/go
go test -v ./...
```
