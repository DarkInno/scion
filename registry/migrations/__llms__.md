# migrations module

Zero-dependency Go SQL migration runner. Copy `src/go/*.go` into `internal/migrations`. Use `New(fsys, Options{Dir: "migrations"})`, then call `Up(ctx, db)`, `Down(ctx, db, steps)`, or `Status(ctx, db)`. Migration files must be named `YYYYMMDDHHMMSS_name.up.sql` with optional paired `.down.sql`. Validates filenames, table names, size limits, null bytes, and path traversal. Use `DollarPlaceholder` for PostgreSQL.
