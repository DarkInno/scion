# Validation Module

Chainable validation builder for request DTOs and HTTP handlers.

## What's Included

- Generic validation builder
- Field-level rules
- Structured validation errors
- HTTP middleware
- CRLF and null-byte rejection rules
- Length and regex protections

## Quick Copy

```bash
cp -r registry/validation/src/go/*.go yourproject/internal/validation/
```

Or with the Scion CLI:

```bash
scion add validation --to internal/validation
```

## Usage

```go
rules := validation.New().
	Field("email").Required().Email().Length(1, 255).
	Field("name").Required().Length(2, 100)

if errs := rules.ValidateJSON(r); errs.HasErrors() {
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{"errors": errs})
	return
}
```

## File Reference

| File | Purpose |
|------|---------|
| `validator.go` | Builder and validation execution |
| `rules.go` | Built-in validation rules |
| `errors.go` | Error types and formatting |
| `middleware.go` | HTTP JSON validation middleware |
| `pentest_test.go` | Malicious input tests |

## Tests

```bash
cd registry/validation/src/go
go test -v ./...
```
