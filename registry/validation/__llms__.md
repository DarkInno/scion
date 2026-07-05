# Validation

Zero-dependency Go validation module. Copy `src/go/*.go` into `internal/validation`. Provides generic builder-style validation for DTO structs, field rules, structured errors, and HTTP middleware. Use max-length rules on all user strings and add CRLF/null-byte rejection where strings may flow into headers, logs, paths, URLs, or SQL filters. Regex rules rely on Go RE2 semantics.
