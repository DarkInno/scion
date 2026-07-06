# Database Module

`database/sql` setup, transaction helpers, and safe query fragments.

## What's Included

- Bounded dynamic connection pool defaults
- Dynamic pool sizing based on CPU and IO capacity
- `FromEnv()` for `DATABASE_*` settings
- Startup ping with timeout and DSN-safe errors
- `WithinTx` helper with rollback on error or panic
- `DBTX` interface for repositories
- Whitelisted `WHERE` and `ORDER BY` fragment builders
- Maximum 32 filter conditions per builder/query

## Quick Copy

```bash
cp -r registry/database/src/go/* yourproject/internal/database/
```

## Usage

```go
opts := database.FromEnv()
db, err := database.Open(ctx, opts)
if err != nil {
    return err
}
defer db.Close()

err = database.WithinTx(ctx, db, nil, func(ctx context.Context, tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, "UPDATE users SET active = ? WHERE id = ?", true, 123)
    return err
})
```

## Safe Fragments

```go
columns := database.ColumnMap{
    "email": "users.email",
    "created_at": "users.created_at",
}

where, args, err := database.WhereEqual(
    map[string]string{"email": "ada@example.com"},
    columns,
    database.DollarPlaceholder,
)
order, err := database.OrderBy("-created_at", columns)
```

## Dynamic Pool Sizing

`Defaults()` uses a balanced dynamic pool based on `GOMAXPROCS`. For IO-heavy
services, opt into a larger bounded pool:

```go
opts := database.ApplyPoolStrategy(database.FromEnv(), database.PoolStrategy{
    Profile:       database.PoolIOHeavy,
    CPUCores:      8,
    IOParallelism: 4,
    MaxOpenLimit:  96,
})
```

Environment knobs:

| Variable | Purpose |
|----------|---------|
| `DATABASE_POOL_PROFILE` | `conservative`, `balanced`, or `io-heavy` |
| `DATABASE_POOL_CPU_CORES` | CPU cores used by the formula |
| `DATABASE_IO_PARALLELISM` | extra independent IO capacity |
| `DATABASE_POOL_MAX_OPEN_LIMIT` | hard cap for derived max open connections |

## Security Features

- Rejects CRLF and null bytes in string inputs
- Enforces length limits on driver names, DSNs, fields, and values
- Requires whitelist mapping for SQL identifiers
- Keeps filter values parameterized
- Caps generated filter conditions to avoid unbounded memory growth
- Does not include DSNs in open or ping errors

## Tests

```bash
cd registry/database/src/go
go test -v ./...
```
