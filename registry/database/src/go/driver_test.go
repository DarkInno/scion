package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const testDriverName = "sciontest"

var registerTestDriver sync.Once

type txCounters struct {
	commits   atomic.Int64
	rollbacks atomic.Int64
}

var testTxCounters txCounters

func ensureTestDriver() {
	registerTestDriver.Do(func() {
		sql.Register(testDriverName, testDriver{})
	})
}

func resetTxCounters() {
	testTxCounters.commits.Store(0)
	testTxCounters.rollbacks.Store(0)
}

type testDriver struct{}

func (testDriver) Open(name string) (driver.Conn, error) {
	if strings.Contains(name, "open-error") {
		return nil, errors.New("open failed for " + name)
	}
	return &testConn{dsn: name}, nil
}

type testConn struct {
	dsn string
}

func (c *testConn) Prepare(string) (driver.Stmt, error) {
	return testStmt{}, nil
}

func (c *testConn) Close() error {
	return nil
}

func (c *testConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *testConn) Ping(ctx context.Context) error {
	if strings.Contains(c.dsn, "ping-error") {
		return errors.New("ping failed for " + c.dsn)
	}
	if strings.Contains(c.dsn, "slow") {
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
	return nil
}

func (c *testConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	if strings.Contains(c.dsn, "begin-error") {
		return nil, errors.New("begin failed")
	}
	return &testTx{dsn: c.dsn}, nil
}

type testTx struct {
	dsn string
}

func (tx *testTx) Commit() error {
	testTxCounters.commits.Add(1)
	if strings.Contains(tx.dsn, "commit-error") {
		return errors.New("commit failed")
	}
	return nil
}

func (tx *testTx) Rollback() error {
	testTxCounters.rollbacks.Add(1)
	if strings.Contains(tx.dsn, "rollback-error") {
		return errors.New("rollback failed")
	}
	return nil
}

type testStmt struct{}

func (testStmt) Close() error {
	return nil
}

func (testStmt) NumInput() int {
	return -1
}

func (testStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

func (testStmt) Query([]driver.Value) (driver.Rows, error) {
	return emptyRows{}, nil
}

type emptyRows struct{}

func (emptyRows) Columns() []string {
	return nil
}

func (emptyRows) Close() error {
	return nil
}

func (emptyRows) Next([]driver.Value) error {
	return io.EOF
}
