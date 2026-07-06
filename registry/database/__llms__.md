# Database

Zero-dependency Go `database/sql` helper module. Copy `src/go/*.go` into `internal/database`; callers import their own SQL driver. Provides `Options`, dynamic pool `PoolStrategy`, `Defaults`, `FromEnv`, `Open`, `WithinTx`, `DBTX`, whitelisted `OrderBy`/`WhereEqual`, `Builder`, and `?`/`$n` placeholders. Rejects CRLF/null bytes and overlong strings, caps filter conditions, never concatenates untrusted filter/sort input into SQL, and does not include DSNs in open/ping errors.
