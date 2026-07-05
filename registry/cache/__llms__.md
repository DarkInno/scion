# Cache

Zero-dependency Go cache module. Copy `src/go/*.go` into `internal/cache`; do not import Scion as a library. Provides generic `Store[V]` plus `MemoryCache[V]` with TTL expiration, LRU eviction, max entry bounds, counters, and clean shutdown. Keys reject empty strings, strings over 256 bytes, CRLF, and null bytes. Always call `Close()` on long-lived caches to stop the cleanup goroutine.
