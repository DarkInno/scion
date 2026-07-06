package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// DBTX is the common subset implemented by *sql.DB and *sql.Tx. Repository
// methods can accept this interface to run either inside or outside a
// transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// Beginner is implemented by *sql.DB and allows tests or adapters to provide a
// transaction-capable database handle.
type Beginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// WithinTx runs fn inside a transaction. It commits when fn returns nil,
// rolls back when fn returns an error, and also rolls back before re-panicking
// if fn panics.
func WithinTx(ctx context.Context, db Beginner, opts *sql.TxOptions, fn func(context.Context, *sql.Tx) error) (err error) {
	if ctx == nil {
		return errors.New("database: context is nil")
	}
	if db == nil {
		return errors.New("database: transaction beginner is nil")
	}
	if fn == nil {
		return errors.New("database: transaction function is nil")
	}

	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return errors.New("database: begin transaction failed")
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("database: transaction failed: %w; rollback failed: %v", err, rbErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return errors.New("database: commit transaction failed")
	}
	return nil
}
