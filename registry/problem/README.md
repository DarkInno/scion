# Problem Module

RFC 9457-style HTTP API problem responses with safe defaults.

## What's Included

- `Problem` and `InvalidParam` JSON types
- `Write` helper for `application/problem+json`
- `Handler` adapter for handlers that return `error`
- Panic recovery middleware
- Request ID extension support
- CRLF, null byte, length, and validation-error count protection

## Quick Copy

```bash
cp -r registry/problem/src/go/*.go yourproject/internal/problem/
```

Or with the Scion CLI:

```bash
scion add problem --to internal/problem
```

## Usage

```go
http.Handle("/users", problem.Handler(func(w http.ResponseWriter, r *http.Request) error {
    return problem.Error(http.StatusNotFound, "User not found", "no user matched the request")
}))
```

Validation errors:

```go
problem.Write(w, r, problem.Validation([]problem.InvalidParam{
    {Detail: "must be a valid email", Pointer: "#/email"},
}))
```

## File Reference

| File | Purpose |
|------|---------|
| `config.go` | Options, defaults, env loading |
| `problem.go` | Problem types and sanitization |
| `handler.go` | JSON writer, error handler, recovery middleware |
| `pentest_test.go` | Attack-scenario tests |

## Tests

```bash
cd registry/problem/src/go
go test -v ./...
```
