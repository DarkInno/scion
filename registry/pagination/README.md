# Pagination Module

Offset and cursor pagination helpers for HTTP APIs.

## What's Included

- Offset/limit parsing
- Cursor parsing and validation
- Response envelope helpers
- HTTP middleware for request-scoped params
- Default and maximum limit enforcement

## Quick Copy

```bash
cp -r registry/pagination/src/go/*.go yourproject/internal/pagination/
```

Or with the Scion CLI:

```bash
scion add pagination --to internal/pagination
```

## Usage

```go
opts := pagination.Defaults()
pager := pagination.NewOffsetPaginator[User](opts)
params := pager.Parse(r)
resp := pager.Paginate(items, total, params)
_ = json.NewEncoder(w).Encode(resp)
```

## File Reference

| File | Purpose |
|------|---------|
| `config.go` | Options, defaults, environment loading |
| `offset.go` | Offset pagination parsing |
| `cursor.go` | Cursor encoding and parsing |
| `middleware.go` | Request context middleware |
| `response.go` | Response envelopes |
| `pentest_test.go` | Input abuse tests |

## Tests

```bash
cd registry/pagination/src/go
go test -v ./...
```
