# crud module

Generic CRUD endpoints with pagination, filtering, and sorting.

## Languages
- Go 1.22+ (src/go/)
- Python 3.12+ (src/python/)

## Config
- DB_URL — database connection
- DEFAULT_PAGE_SIZE — default 20
- MAX_PAGE_SIZE — default 100

## Adapt (Go)
- Define your entity struct, embed `crud.BaseEntity`
- Implement `EntityStore[T]` interface
- Call `crud.NewHandler(store, cfg)` and `RegisterRoutes(mux, "/api/v1/entity")`

## Adapt (Python)
- src/models/<entity>.py — define your entity model
- src/schemas/<entity>.py — Pydantic request/response schemas
- src/routes.py — register CRUD routes with your model

## Deps (Go)
joho/godotenv

## Deps (Python)
SQLAlchemy 2.0, Pydantic
