package migrations

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

var (
	fakeDriverOnce sync.Once
	fakeStatesMu   sync.Mutex
	fakeStates     = map[string]*fakeState{}
)

type fakeState struct {
	mu        sync.Mutex
	applied   map[int64]AppliedMigration
	execs     []string
	failOn    string
	commits   int
	rollbacks int
}

type fakeDriver struct{}
type fakeConn struct{ state *fakeState }
type fakeTx struct{ state *fakeState }
type fakeRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}
type fakeResult int64

func openFakeDB(t *testing.T) (*sql.DB, *fakeState) {
	t.Helper()
	fakeDriverOnce.Do(func() {
		sql.Register("scion_migrations_fake", fakeDriver{})
	})
	name := fmt.Sprintf("db_%d", time.Now().UnixNano())
	state := &fakeState{applied: map[int64]AppliedMigration{}}
	fakeStatesMu.Lock()
	fakeStates[name] = state
	fakeStatesMu.Unlock()
	db, err := sql.Open("scion_migrations_fake", name)
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		fakeStatesMu.Lock()
		delete(fakeStates, name)
		fakeStatesMu.Unlock()
	})
	return db, state
}

func (fakeDriver) Open(name string) (driver.Conn, error) {
	fakeStatesMu.Lock()
	defer fakeStatesMu.Unlock()
	state := fakeStates[name]
	if state == nil {
		return nil, errors.New("unknown fake db")
	}
	return &fakeConn{state: state}, nil
}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not implemented")
}
func (c *fakeConn) Close() error              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error) { return &fakeTx{state: c.state}, nil }

func (c *fakeConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &fakeTx{state: c.state}, nil
}

func (c *fakeConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	if c.state.failOn != "" && strings.Contains(query, c.state.failOn) {
		return nil, errors.New("forced execution failure")
	}
	upper := strings.ToUpper(strings.TrimSpace(query))
	switch {
	case strings.HasPrefix(upper, "CREATE TABLE IF NOT EXISTS"):
		return fakeResult(0), nil
	case strings.HasPrefix(upper, "INSERT INTO"):
		version := args[0].Value.(int64)
		c.state.applied[version] = AppliedMigration{
			Version:   version,
			Name:      args[1].Value.(string),
			Checksum:  args[2].Value.(string),
			AppliedAt: sql.NullTime{Time: args[3].Value.(time.Time), Valid: true},
		}
		return fakeResult(1), nil
	case strings.HasPrefix(upper, "DELETE FROM"):
		delete(c.state.applied, args[0].Value.(int64))
		return fakeResult(1), nil
	default:
		c.state.execs = append(c.state.execs, query)
		return fakeResult(1), nil
	}
}

func (c *fakeConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT VERSION") {
		return nil, errors.New("unexpected query")
	}
	versions := make([]int64, 0, len(c.state.applied))
	for version := range c.state.applied {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	values := make([][]driver.Value, 0, len(versions))
	for _, version := range versions {
		row := c.state.applied[version]
		values = append(values, []driver.Value{row.Version, row.Name, row.Checksum, row.AppliedAt.Time})
	}
	return &fakeRows{
		columns: []string{"version", "name", "checksum", "applied_at"},
		values:  values,
	}, nil
}

func (tx *fakeTx) Commit() error {
	tx.state.mu.Lock()
	defer tx.state.mu.Unlock()
	tx.state.commits++
	return nil
}

func (tx *fakeTx) Rollback() error {
	tx.state.mu.Lock()
	defer tx.state.mu.Unlock()
	tx.state.rollbacks++
	return nil
}

func (r *fakeRows) Columns() []string { return r.columns }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r fakeResult) RowsAffected() (int64, error) {
	return int64(r), nil
}

func TestUpAppliesUnappliedMigrations(t *testing.T) {
	db, state := openFakeDB(t)
	fsys := fstest.MapFS{
		"20260101000001_add_users.up.sql":   {Data: []byte("CREATE TABLE users(id BIGINT);")},
		"20260101000001_add_users.down.sql": {Data: []byte("DROP TABLE users;")},
	}
	m, err := New(fsys, Options{Now: func() time.Time { return time.Unix(1, 0) }})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := m.Up(context.Background(), db); err != nil {
		t.Fatalf("up: %v", err)
	}
	if len(state.applied) != 1 || len(state.execs) != 1 {
		t.Fatalf("unexpected state: %+v", state)
	}
	if state.commits != 1 || state.rollbacks != 0 {
		t.Fatalf("transaction counts: commits=%d rollbacks=%d", state.commits, state.rollbacks)
	}
}

func TestUpDetectsChecksumMismatch(t *testing.T) {
	db, state := openFakeDB(t)
	state.applied[20260101000001] = AppliedMigration{Version: 20260101000001, Name: "add_users", Checksum: "old"}
	fsys := fstest.MapFS{
		"20260101000001_add_users.up.sql": {Data: []byte("CREATE TABLE users(id BIGINT);")},
	}
	m, err := New(fsys)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := m.Up(context.Background(), db); err == nil {
		t.Fatalf("expected checksum mismatch")
	}
}

func TestDownRollsBackLatestMigration(t *testing.T) {
	db, state := openFakeDB(t)
	fsys := fstest.MapFS{
		"20260101000001_add_users.up.sql":     {Data: []byte("CREATE TABLE users(id BIGINT);")},
		"20260101000001_add_users.down.sql":   {Data: []byte("DROP TABLE users;")},
		"20260101000002_add_widgets.up.sql":   {Data: []byte("CREATE TABLE widgets(id BIGINT);")},
		"20260101000002_add_widgets.down.sql": {Data: []byte("DROP TABLE widgets;")},
	}
	m, err := New(fsys)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := m.Up(context.Background(), db); err != nil {
		t.Fatalf("up: %v", err)
	}
	if err := m.Down(context.Background(), db, 1); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, ok := state.applied[20260101000002]; ok {
		t.Fatalf("latest migration still applied")
	}
	if _, ok := state.applied[20260101000001]; !ok {
		t.Fatalf("older migration was removed")
	}
}

func TestStatusReportsMissingAndMismatch(t *testing.T) {
	db, state := openFakeDB(t)
	state.applied[20260101000002] = AppliedMigration{Version: 20260101000002, Name: "missing", Checksum: "abc"}
	state.applied[20260101000001] = AppliedMigration{Version: 20260101000001, Name: "add_users", Checksum: "old"}
	fsys := fstest.MapFS{
		"20260101000001_add_users.up.sql": {Data: []byte("CREATE TABLE users(id BIGINT);")},
	}
	m, err := New(fsys)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	statuses, err := m.Status(context.Background(), db)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("len = %d", len(statuses))
	}
	if !statuses[0].ChecksumMismatch || !statuses[1].Missing {
		t.Fatalf("unexpected statuses: %+v", statuses)
	}
}

func TestStatusOmitsAppliedAtForUnappliedMigration(t *testing.T) {
	db, _ := openFakeDB(t)
	fsys := fstest.MapFS{
		"20260101000001_add_users.up.sql": {Data: []byte("CREATE TABLE users(id BIGINT);")},
	}
	m, err := New(fsys)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	statuses, err := m.Status(context.Background(), db)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	data, err := json.Marshal(statuses[0])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "applied_at") {
		t.Fatalf("applied_at should be omitted for unapplied migration: %s", data)
	}
}

func TestAppliedRowsRespectsMaxMigrations(t *testing.T) {
	db, state := openFakeDB(t)
	state.applied[20260101000001] = AppliedMigration{Version: 20260101000001, Name: "one", Checksum: "a"}
	state.applied[20260101000002] = AppliedMigration{Version: 20260101000002, Name: "two", Checksum: "b"}
	fsys := fstest.MapFS{
		"20260101000001_one.up.sql": {Data: []byte("SELECT 1;")},
	}
	m, err := New(fsys, Options{MaxMigrations: 1})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := m.Status(context.Background(), db); err == nil {
		t.Fatalf("expected applied row limit error")
	}
}
