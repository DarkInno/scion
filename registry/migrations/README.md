# Migrations Module

Standard-library-only SQL migration runner for `database/sql`.

## What's Included

- `fs.FS` migration discovery
- Paired `YYYYMMDDHHMMSS_name.up.sql` / `.down.sql` files
- Schema history table with version, name, checksum, and timestamp
- Transactional `Up`, `Down`, and `Status`
- Safe table-name and filename validation
- Status HTTP handler with panic recovery

## Quick Copy

```bash
cp -r registry/migrations/src/go/*.go yourproject/internal/migrations/
```

Or with the Scion CLI:

```bash
scion add migrations --to internal/migrations
```

## Usage

```go
//go:embed migrations/*.sql
var migrationFS embed.FS

m, err := migrations.New(migrationFS, migrations.Options{
    Dir: "migrations",
})
if err != nil {
    return err
}
if err := m.Up(ctx, db); err != nil {
    return err
}
```

For PostgreSQL-style placeholders:

```go
m, _ := migrations.New(migrationFS, migrations.Options{
    Dir:         "migrations",
    Placeholder: migrations.DollarPlaceholder,
})
```

## File Reference

| File | Purpose |
|------|---------|
| `config.go` | Options, defaults, env loading, placeholder helpers |
| `file.go` | Migration filename parsing, SQL loading, checksums |
| `migrator.go` | `Up`, `Down`, `Status`, and schema history bookkeeping |
| `handler.go` | HTTP status handler |
| `pentest_test.go` | Attack-scenario tests |

## Tests

```bash
cd registry/migrations/src/go
go test -v ./...
```
