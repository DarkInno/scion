# Validation Module

Chainable request validation builder with security-first design.

## What's Included

- Chainable validation builder
- Common validation rules
- Regex DoS prevention (RE2 engine)
- Null byte and CRLF rejection
- Panic recovery
- HTTP middleware

## Quick Copy

```bash
cp -r registry/validation/src/go/* yourproject/internal/validation/
```

## Usage

### Define Validation Rules

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

rules := validation.For[CreateUserRequest]().
    Field("name").
        Required().
        MinLength(2).
        MaxLength(100).
        CRLF().
        NullByte().
    Field("email").
        Required().
        Email().
        MaxLength(255).
    Field("age").
        Required().
        Min(0).
        Max(150)
```

### Validate Request

```go
handler := func(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    if errs := rules.Validate(req); len(errs) > 0 {
        // Return validation errors
        w.WriteHeader(http.StatusBadRequest)
        _ = json.NewEncoder(w).Encode(map[string]any{"errors": errs})
        return
    }
    
    // Process valid request
}
```

### Use Middleware

```go
handler := validation.Middleware(rules)(handler)
```

## Available Rules

| Rule | Description |
|------|-------------|
| `Required()` | Field must not be empty |
| `MinLength(n)` | Minimum string length |
| `MaxLength(n)` | Maximum string length |
| `Min(n)` | Minimum numeric value |
| `Max(n)` | Maximum numeric value |
| `Email()` | Valid email format |
| `URL()` | Valid URL format |
| `Pattern(regex)` | Regex match (RE2 engine) |
| `In(values...)` | Value in allowed list |
| `CRLF()` | Reject CRLF injection |
| `NullByte()` | Reject null bytes |

## File Reference

| File | Purpose |
|------|---------|
| `validator.go` | Core validation logic |
| `rules.go` | Validation rules |
| `errors.go` | Error types |
| `middleware.go` | HTTP middleware |

## Security Features

- Regex DoS prevention (RE2 engine, no backtracking)
- Null byte rejection
- CRLF injection prevention
- Panic recovery in validation

## Tests

```bash
cd registry/validation/src/go
go test -v ./...
```
