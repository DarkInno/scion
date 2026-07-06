# Migrations Module

Standard-library-only SQL migration runner for `database/sql`.

## Features

- Loads `YYYYMMDDHHMMSS_name.up.sql` and optional `.down.sql` files from `fs.FS`
- Records version, name, checksum, and timestamp in `schema_migrations`
- Applies migrations transactionally by default
- Supports `Up`, `Down`, and `Status`
- Includes a status HTTP handler with panic recovery

## Usage

```go
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

Use `migrations.DollarPlaceholder` for PostgreSQL-style placeholders.

## Security

- Rejects CRLF and null bytes in config/file names
- Rejects path traversal in migration directory and names
- Validates schema history table identifiers
- Caps migration count and SQL file size

## Copy

```bash
scion add migrations --to internal/migrations
```
