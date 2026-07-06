# Problem Details Module

RFC 9457-style API error responses for `net/http`.

## Features

- `Problem` and `InvalidParam` JSON types
- `application/problem+json` writer
- Handler adapter for functions returning `error`
- Panic recovery middleware
- Optional safe request ID extension

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

## Security

- Rejects CRLF and null bytes in response fields
- Truncates long details
- Hides unknown internal errors behind generic 500 responses
- Limits validation error count

## Copy

```bash
scion add problem --to internal/problem
```
