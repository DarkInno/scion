package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Open validates Options, opens a database/sql handle, applies pool settings,
// and verifies connectivity with PingContext. Returned errors do not include
// the DSN.
func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	if ctx == nil {
		return nil, errors.New("database: context is nil")
	}
	opts = opts.normalize()
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	db, err := sql.Open(opts.DriverName, opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("database: open driver %q: %w", opts.DriverName, err)
	}
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(opts.ConnMaxLifetime)
	db.SetConnMaxIdleTime(opts.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, opts.PingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, errors.New("database: ping failed")
	}
	return db, nil
}
