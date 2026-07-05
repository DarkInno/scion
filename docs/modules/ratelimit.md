# Rate Limit Module

Rate limiting algorithms with memory exhaustion protection.

## What's Included

- **Fixed Window** — simple counter-based limiting
- **Sliding Window** — smoother rate limiting
- **Token Bucket** — burst-friendly limiting
- Memory exhaustion protection with LRU eviction
- HTTP middleware

## Quick Copy

```bash
cp -r registry/ratelimit/src/go/* yourproject/internal/ratelimit/
```

## Usage

### Fixed Window

```go
limiter := ratelimit.NewFixedWindow(100, time.Minute)
handler := ratelimit.Middleware(limiter)(handler)
```

### Sliding Window

```go
limiter := ratelimit.NewSlidingWindow(100, time.Minute)
handler := ratelimit.Middleware(limiter)(handler)
```

### Token Bucket

```go
limiter := ratelimit.NewTokenBucket(100, 10) // 100 tokens/sec, burst 10
handler := ratelimit.Middleware(limiter)(handler)
```

### Custom Key Function

```go
handler := ratelimit.Middleware(limiter, ratelimit.WithKeyFunc(func(r *http.Request) string {
    return r.Header.Get("X-API-Key")
}))(handler)
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `maxRequests` | Maximum requests per window | Required |
| `window` | Time window duration | Required |
| `burst` | Burst capacity (token bucket) | Required |
| `maxBuckets` | Maximum tracked keys | 10000 |

## File Reference

| File | Purpose |
|------|---------|
| `fixed_window.go` | Fixed window algorithm |
| `sliding_window.go` | Sliding window algorithm |
| `token_bucket.go` | Token bucket algorithm |
| `store.go` | In-memory store with LRU eviction |
| `middleware.go` | HTTP middleware |

## Security Features

- Memory exhaustion protection with max buckets
- LRU eviction when limit reached
- Key length limit prevents abuse

## Tests

```bash
cd registry/ratelimit/src/go
go test -v ./...
```
