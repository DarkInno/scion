# Cache Module

Generic TTL + LRU in-memory cache with background cleanup.

## What's Included

- Generic cache with type safety
- TTL (Time To Live) support
- LRU (Least Recently Used) eviction
- Background cleanup goroutine
- Memory exhaustion protection

## Quick Copy

```bash
cp -r registry/cache/src/go/* yourproject/internal/cache/
```

## Usage

### Basic Cache

```go
// Create cache with 5-minute TTL and max 1000 entries
c := cache.New[string, User](5*time.Minute, 1000)

// Set value
c.Set("user:123", user)

// Get value
user, ok := c.Get("user:123")

// Delete value
c.Delete("user:123")
```

### With Custom TTL

```go
c.SetWithTTL("key", value, 10*time.Minute)
```

### Cache Stats

```go
stats := c.Stats()
fmt.Printf("Hits: %d, Misses: %d, Size: %d", stats.Hits, stats.Misses, stats.Size)
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ttl` | Time to live for entries | Required |
| `maxEntries` | Maximum entries in cache | 1000 |
| `cleanupInterval` | Background cleanup interval | ttl/2 |

## File Reference

| File | Purpose |
|------|---------|
| `memory.go` | Cache implementation |
| `store.go` | Store interface |
| `pentest_test.go` | Security tests |

## Security Features

- Memory exhaustion protection with max entries
- LRU eviction when limit reached
- Background cleanup prevents memory leaks
- Goroutine leak prevention

## Tests

```bash
cd registry/cache/src/go
go test -v ./...
```
