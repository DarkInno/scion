# Pagination Module

Offset/limit and cursor pagination with security-first design.

## What's Included

- **Offset pagination** — traditional page-based
- **Cursor pagination** — efficient for large datasets
- Base64 cursor validation
- Negative offset clamping
- Max limit enforcement

## Quick Copy

```bash
cp -r registry/pagination/src/go/* yourproject/internal/pagination/
```

## Usage

### Offset Pagination

```go
handler := pagination.OffsetHandler(pagination.OffsetConfig{
    DefaultLimit: 20,
    MaxLimit: 100,
})

// In your handler
func listUsers(w http.ResponseWriter, r *http.Request) {
    params := pagination.GetOffsetParams(r)
    // params.Page, params.Limit
    
    users, total, err := store.List(r.Context(), params)
    // Return paginated response
}
```

### Cursor Pagination

```go
handler := pagination.CursorHandler(pagination.CursorConfig{
    DefaultLimit: 20,
    MaxLimit: 100,
})

// In your handler
func listUsers(w http.ResponseWriter, r *http.Request) {
    params := pagination.GetCursorParams(r)
    // params.Cursor, params.Limit
    
    users, nextCursor, err := store.List(r.Context(), params)
    // Return paginated response with next_cursor
}
```

## Response Format

### Offset

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

### Cursor

```json
{
  "data": [...],
  "pagination": {
    "next_cursor": "eyJpZCI6MTIzfQ==",
    "has_more": true
  }
}
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `DefaultLimit` | Default page size | 20 |
| `MaxLimit` | Maximum page size | 100 |

## File Reference

| File | Purpose |
|------|---------|
| `offset.go` | Offset pagination |
| `cursor.go` | Cursor pagination |
| `config.go` | Configuration |
| `response.go` | Response types |
| `middleware.go` | HTTP middleware |

## Security Features

- Cursor base64 validation
- Negative offset clamping
- Max limit enforcement
- Input validation

## Tests

```bash
cd registry/pagination/src/go
go test -v ./...
```
