# Security Design

Scion modules are built with security-first principles. Every module implements these non-negotiable security requirements.

## Core Principles

### 1. Never Trust Client Input

All user inputs are validated, sanitized, and length-limited before processing.

### 2. Defense in Depth

Multiple layers of protection: input validation, rate limiting, output escaping.

### 3. Fail Securely

Errors return generic messages. No internal details leaked to clients.

## Security Features

### Input Validation

- **CRLF injection prevention** — reject `\r\n` in all user inputs (headers, URLs, names)
- **Null byte rejection** — reject `\x00` in all string inputs
- **Length limits** — all user-supplied strings have max length checks (128-1024 chars depending on context)

### Memory Protection

- **Memory exhaustion protection** — maps/slices with unbounded growth MUST have `maxBuckets` or `maxEntries` limits + LRU eviction
- **Rate limiting** — fixed window, sliding window, and token bucket algorithms

### Path Security

- **Path traversal prevention** — use `filepath.Base()` + reject `..` in all file operations
- **No X-Forwarded-For trust** — `ClientIP()` returns `r.RemoteAddr` only; XFF is client-controlled and spoofable

### SQL Security

- **Parameterized queries** — never concatenate user input into SQL

### HTTP Security

- **Panic recovery** — all HTTP handlers must recover from panics
- **Body size limits** — prevent large payload attacks
- **Timeout** — prevent slowloris attacks

## Module-Specific Security

| Module | Security Features |
|--------|-------------------|
| Auth | Rate limiting, user enumeration prevention, JTI, aud/iss validation |
| CRUD | Sort/filter whitelist, SQL injection prevention, pagination ceiling |
| Middleware | CRLF injection prevention, trusted proxy, body size limit |
| RBAC | Wildcard permissions, cycle detection, hierarchy inheritance |
| Rate Limit | Memory exhaustion protection, LRU eviction, key length limit |
| Validation | Regex DoS prevention (RE2), null byte/CRLF rejection, panic recovery |
| File Upload | Magic bytes validation, path traversal prevention, size limit |
| Health | SSRF protection (private IP rejection), CRLF injection prevention |
| Cache | Background cleanup, goroutine leak prevention, max entries limit |
| Pagination | Cursor base64 validation, negative offset clamp, max limit enforcement |
| Mail | Header injection prevention, XSS escaping, attachment sanitization |

## Security Checklist

When adapting any module, ensure:

- [ ] All user inputs are validated before processing
- [ ] Length limits are enforced on all strings
- [ ] Rate limiting is configured for sensitive endpoints
- [ ] SQL queries use parameterized statements
- [ ] File paths are sanitized with `filepath.Base()`
- [ ] Error messages don't leak internal details
- [ ] HTTP handlers recover from panics

## Testing Security

Every module includes `pentest_test.go` with attack-scenario test cases:

```bash
cd registry/<module>/src/go
go test -v -run Pentest ./...
```

Test coverage includes:
- CRLF injection attempts
- Null byte injection
- Path traversal attempts
- SQL injection attempts
- Memory exhaustion attempts
- Rate limiting bypass attempts
