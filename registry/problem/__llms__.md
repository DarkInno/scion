# problem module

Zero-dependency Go module for RFC 9457-style API errors. Copy `src/go/*.go` into `internal/problem`. Use `Write(w, r, Problem)`, `Error(status,title,detail)`, `Handler(func(w,r) error)`, and `Recoverer()`. Responses use `application/problem+json` and sanitize type/title/detail/instance/request_id/validation errors for CRLF, null bytes, length limits, and internal-error leakage.
