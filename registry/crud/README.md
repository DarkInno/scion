# CRUD Module

Generic CRUD operations with pagination, filtering, and sorting.

## What's Included

- `POST /` — Create
- `GET /` — List (with pagination, filtering, sorting)
- `GET /:id` — Read one
- `PUT /:id` — Update
- `DELETE /:id` — Delete
- Sort field whitelist (prevents SQL injection)
- Filter field whitelist (prevents SQL injection)

## Quick Copy

### Go

```bash
cp -r registry/crud/src/go/* src/crud/
cp -r registry/crud/examples/gin/* src/crud/
```

### Python

```bash
cp -r registry/crud/src/python/* src/crud/
cp -r registry/crud/examples/fastapi/* src/crud/
```

## Adaptation Guide (Go)

1. **Entity model** — define your struct and embed `crud.BaseEntity`
2. **Store layer** — implement `crud.EntityStore[T]` with your database layer
3. **Configuration** — set environment variables:
   - `DB_URL` — required
   - `DEFAULT_PAGE_SIZE` — default 20, cannot exceed `MAX_PAGE_SIZE`
   - `MAX_PAGE_SIZE` — default 100, hard ceiling for all list requests
4. **Sort validation** (required) — call `handler.WithSortValidator(fn)` to whitelist allowed sort fields. Without this, ALL sort parameters are rejected
5. **Filter fields** (optional) — call `handler.WithFilterFields(allowed)` to whitelist allowed filter keys. Without this, all query parameters except offset/limit/sort are accepted as filters
6. **Routes** — call `handler.RegisterRoutes(mux, "/api/v1/your-entity")`. Returns error if basePath is invalid

## File Reference (Go)

| File | Purpose |
|------|---------|
| `config.go` | Env var loading, pagination defaults |
| `models.go` | BaseEntity, PaginatedResponse[T], ListParams, SortField, filter utilities |
| `handlers.go` | Generic CRUD HTTP handlers (Go 1.18+ generics), sort/filter whitelist |
| `routes.go` | Route registration, ValidateRoutePrefix |

## Tests

Every file has corresponding `*_test.go` coverage:

```bash
cd registry/crud/src/go
go test -v ./...
```

Test coverage includes:
- Config validation (page size defaults, max ceiling, capping)
- Pagination and sort parsing (negative offset clamping, limit capping, descending prefix)
- Filter sanitization and validation
- CRUD HTTP handlers (create, get, list, update, delete) with mock generic store
- Sort/filter whitelist enforcement
- Nil slice protection (ensures `[]` not `null` in JSON)
- Route registration and validation

## Query Parameters

- `?offset=0&limit=20` — Pagination
- `?sort=-created_at` — Sorting (prefix `-` for descending). Must be in sort whitelist
- `?name=foo&status=active` — Filtering by exact match. Must be in filter whitelist if configured

## Security Checklist

When adapting this module, ensure:
- [ ] `WithSortValidator` is configured with your schema's column names
- [ ] `WithFilterFields` is configured if you expose filtering (prevents arbitrary column filtering)
- [ ] `MAX_PAGE_SIZE` is set to a reasonable ceiling for your data volume

## Example Usage

See `examples/gin/` for a template Go project with an in-memory Product store.
