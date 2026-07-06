# Database Module

Small `database/sql` helpers for connection setup, transaction wrapping, and
safe query fragments.

## What's Included

- Bounded dynamic `database/sql` pool defaults
- Dynamic pool sizing based on CPU and IO capacity
- Environment-based configuration
- Startup ping with timeout
- Transaction helper with rollback on error or panic
- Whitelisted `WHERE` and `ORDER BY` builders
- Filter condition cap to prevent unbounded slice growth
- `?` and PostgreSQL `$n` placeholders

## Quick Copy

```bash
cp -r registry/database/src/go/*.go yourproject/internal/database/
```

Or with the Scion CLI:

```bash
scion add database --to internal/database
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

Safe fragments:

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
_ = where
_ = args
_ = order
_ = err
```

Dynamic pool sizing:

```go
opts := database.FromEnv() // balanced dynamic pool by default
opts = database.ApplyPoolStrategy(opts, database.PoolStrategy{
    Profile:       database.PoolIOHeavy,
    CPUCores:      8,
    IOParallelism: 4,
    MaxOpenLimit:  96,
})
```

`Builder` is intended for one query. If you reuse it, call `Reset()` to drop
old argument references.

## Environment

| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_DRIVER` | `database/sql` driver name | required |
| `DATABASE_DSN` | driver-specific DSN | required |
| `DATABASE_MAX_OPEN_CONNS` | explicit max open override | dynamic |
| `DATABASE_MAX_IDLE_CONNS` | explicit max idle override | dynamic |
| `DATABASE_POOL_PROFILE` | `conservative`, `balanced`, `io-heavy` | `balanced` |
| `DATABASE_POOL_CPU_CORES` | CPU cores for dynamic sizing | `GOMAXPROCS` |
| `DATABASE_IO_PARALLELISM` | extra IO capacity for dynamic sizing | 1 |
| `DATABASE_POOL_MAX_OPEN_LIMIT` | hard cap for dynamic max open | profile default |
| `DATABASE_CONN_MAX_LIFETIME` | max connection lifetime | 30m |
| `DATABASE_CONN_MAX_IDLE_TIME` | max idle time | 10m |
| `DATABASE_PING_TIMEOUT` | startup ping timeout | 5s |

## File Reference

| File | Purpose |
|------|---------|
| `config.go` | Options, defaults, environment loading |
| `profile.go` | Dynamic connection pool sizing |
| `open.go` | Open and ping a configured `*sql.DB` |
| `transaction.go` | Transaction helper and DBTX interface |
| `query.go` | Whitelisted SQL fragment helpers |
| `pentest_test.go` | Security and abuse-case tests |

## Tests

The standalone `go.mod` uses `module scion-database` instead of
`module database` to avoid colliding with Go's standard-library `database`
import tree on Linux runners. The package name remains `database`.

```bash
cd registry/database/src/go
go test -v ./...
```
