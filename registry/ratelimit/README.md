# Rate Limit Module

Fixed window, sliding window, and token bucket rate limiters with HTTP middleware.

## What's Included

- Fixed window limiter
- Sliding window limiter
- Token bucket limiter
- In-memory store with max bucket bounds and LRU-style eviction
- `net/http` middleware
- Key length and injection validation

## Quick Copy

```bash
cp -r registry/ratelimit/src/go/*.go yourproject/internal/ratelimit/
```

Or with the Scion CLI:

```bash
scion add ratelimit --to internal/ratelimit
```

## Usage

```go
store := ratelimit.NewMemoryStore()
limiter, _ := ratelimit.NewSlidingWindowLimiter(store, 100, time.Minute)
handler := ratelimit.Middleware(limiter, ratelimit.KeyByIP)(next)
```

## File Reference

| File | Purpose |
|------|---------|
| `fixed_window.go` | Fixed window algorithm |
| `sliding_window.go` | Sliding window algorithm |
| `token_bucket.go` | Token bucket algorithm |
| `store.go` | Bounded in-memory state store |
| `middleware.go` | HTTP middleware |
| `pentest_test.go` | Abuse and memory-bound tests |

## Tests

```bash
cd registry/ratelimit/src/go
go test -v ./...
```
