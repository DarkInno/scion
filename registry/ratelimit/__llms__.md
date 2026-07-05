# Ratelimit

Zero-dependency Go rate limiting module. Copy `src/go/*.go` into `internal/ratelimit`. Provides fixed window, sliding window, token bucket, bounded in-memory store, and standard `net/http` middleware. `KeyByIP` uses `r.RemoteAddr` only and ignores `X-Forwarded-For`/`X-Real-IP`. Middleware replaces empty, CRLF, and null-byte keys with `anonymous`, truncates oversized keys, and the store evicts when max buckets is reached.
