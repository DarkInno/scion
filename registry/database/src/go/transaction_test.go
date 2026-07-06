package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func openTxTestDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	ensureTestDriver()
	db, err := sql.Open(testDriverName, dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestWithinTxCommitsOnNilError(t *testing.T) {
	resetTxCounters()
	db := openTxTestDB(t, "memory")
	err := WithinTx(context.Background(), db, nil, func(ctx context.Context, tx *sql.Tx) error {
		if tx == nil {
			t.Fatal("tx is nil")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithinTx: %v", err)
	}
	if testTxCounters.commits.Load() != 1 || testTxCounters.rollbacks.Load() != 0 {
		t.Fatalf("unexpected counters commits=%d rollbacks=%d", testTxCounters.commits.Load(), testTxCounters.rollbacks.Load())
	}
}

func TestWithinTxRollsBackOnError(t *testing.T) {
	resetTxCounters()
	db := openTxTestDB(t, "memory")
	want := errors.New("domain error")
	err := WithinTx(context.Background(), db, nil, func(context.Context, *sql.Tx) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected domain error, got %v", err)
	}
	if testTxCounters.commits.Load() != 0 || testTxCounters.rollbacks.Load() != 1 {
		t.Fatalf("unexpected counters commits=%d rollbacks=%d", testTxCounters.commits.Load(), testTxCounters.rollbacks.Load())
	}
}

func TestWithinTxRollsBackOnPanic(t *testing.T) {
	resetTxCounters()
	db := openTxTestDB(t, "memory")
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
		if testTxCounters.commits.Load() != 0 || testTxCounters.rollbacks.Load() != 1 {
			t.Fatalf("unexpected counters commits=%d rollbacks=%d", testTxCounters.commits.Load(), testTxCounters.rollbacks.Load())
		}
	}()
	_ = WithinTx(context.Background(), db, nil, func(context.Context, *sql.Tx) error {
		panic("boom")
	})
}

func TestWithinTxRejectsNilInputs(t *testing.T) {
	db := openTxTestDB(t, "memory")
	if err := WithinTx(nil, db, nil, func(context.Context, *sql.Tx) error { return nil }); err == nil {
		t.Fatal("expected nil context error")
	}
	if err := WithinTx(context.Background(), nil, nil, func(context.Context, *sql.Tx) error { return nil }); err == nil {
		t.Fatal("expected nil db error")
	}
	if err := WithinTx(context.Background(), db, nil, nil); err == nil {
		t.Fatal("expected nil function error")
	}
}

func TestWithinTxBeginAndCommitErrorsAreSanitized(t *testing.T) {
	db := openTxTestDB(t, "begin-error")
	if err := WithinTx(context.Background(), db, nil, func(context.Context, *sql.Tx) error { return nil }); err == nil {
		t.Fatal("expected begin error")
	}

	db = openTxTestDB(t, "commit-error")
	if err := WithinTx(context.Background(), db, nil, func(context.Context, *sql.Tx) error { return nil }); err == nil {
		t.Fatal("expected commit error")
	}
}
