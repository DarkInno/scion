# Cache Module

Generic TTL + LRU in-memory cache with a small `Store[V]` interface.

## What's Included

- Type-safe generic cache values
- TTL expiration and background cleanup
- LRU eviction with a configurable max entry cap
- Atomic integer counters
- Key validation for CRLF, null bytes, empty strings, and length limits

## Quick Copy

```bash
cp -r registry/cache/src/go/*.go yourproject/internal/cache/
```

Or with the Scion CLI:

```bash
scion add cache --to internal/cache
```

## Usage

```go
c := cache.New[string](cache.WithMaxEntries(1000))
defer c.Close()

_ = c.Set(ctx, "user:123", "Ada", 5*time.Minute)
value, ok := c.Get(ctx, "user:123")
_ = value
_ = ok
```

## File Reference

| File | Purpose |
|------|---------|
| `store.go` | Store interface, key validation, entry and LRU primitives |
| `memory.go` | Concurrency-safe in-memory implementation |
| `pentest_test.go` | Security and abuse-case tests |

## Tests

```bash
cd registry/cache/src/go
go test -v ./...
```
