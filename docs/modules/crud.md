# CRUD Module

Generic CRUD operations with pagination, sort/filter whitelist, and SQL injection prevention.

## What's Included

- Generic CRUD handlers (Create, Read, Update, Delete, List)
- Pagination (offset/limit)
- Sort and filter whitelist
- SQL injection prevention
- Input validation

## Quick Copy

```bash
cp -r registry/crud/src/go/* yourproject/internal/crud/
```

## Adaptation Guide

### 1. Define Your Model

```go
type Product struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Price     float64   `json:"price"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 2. Implement Store Interface

```go
type Store[T any] interface {
    Create(ctx context.Context, entity *T) error
    GetByID(ctx context.Context, id string) (*T, error)
    Update(ctx context.Context, id string, entity *T) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, opts ListOptions) ([]T, int, error)
}
```

### 3. Configure

```go
handler := crud.NewHandler(store, crud.Config{
    MaxPageSize: 100,
    DefaultPageSize: 20,
    SortWhitelist: []string{"name", "created_at"},
    FilterWhitelist: []string{"name", "price"},
})
```

## File Reference

| File | Purpose |
|------|---------|
| `config.go` | Configuration options |
| `models.go` | Generic types and interfaces |
| `handlers.go` | HTTP handlers |
| `routes.go` | Route registration |

## Security Features

- Sort/filter whitelist prevents arbitrary field access
- Parameterized SQL queries
- Input validation on all fields
- Pagination ceiling prevents memory exhaustion

## Tests

```bash
cd registry/crud/src/go
go test -v ./...
```
