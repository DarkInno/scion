# Middleware

Zero-dependency Go `net/http` middleware module. Copy `src/go/*.go` into `internal/middleware`. Exposes standard `func(http.Handler) http.Handler` middleware for chaining, recovery, CORS, structured logging, request IDs, timeouts, body limits, trusted proxies, tracing, and debug guards. Security behavior rejects CRLF/null bytes in headers and request IDs and does not trust client-controlled proxy headers by default.
