# Pagination

Zero-dependency Go pagination module. Copy `src/go/*.go` into `internal/pagination`. Provides offset and cursor parameter parsing, default/max limit enforcement, request-context middleware, and response builders. Inputs reject CRLF/null bytes and oversize cursor/sort values. Cursor helpers use base64 URL encoding with JSON payloads; users own database query integration.
