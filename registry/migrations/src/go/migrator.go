package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"time"
)

// Migrator applies SQL migrations and records applied versions in a schema
// history table.
type Migrator struct {
	fsys fs.FS
	opts Options
}

// AppliedMigration is a row from the schema history table.
type AppliedMigration struct {
	Version   int64
	Name      string
	Checksum  string
	AppliedAt sql.NullTime
}

// Status reports the local file and database state for a migration version.
type Status struct {
	Version          int64      `json:"version"`
	Name             string     `json:"name"`
	Applied          bool       `json:"applied"`
	AppliedAt        *time.Time `json:"applied_at,omitempty"`
	Checksum         string     `json:"checksum,omitempty"`
	ChecksumMismatch bool       `json:"checksum_mismatch,omitempty"`
	Missing          bool       `json:"missing,omitempty"`
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// New returns a migrator over fsys. If opts is omitted, Defaults() is used.
func New(fsys fs.FS, opts ...Options) (*Migrator, error) {
	opt := Defaults()
	if len(opts) > 0 {
		opt = opts[0]
	}
	opt, err := opt.normalize()
	if err != nil {
		return nil, err
	}
	if fsys == nil {
		return nil, errors.New("migrations: fsys is nil")
	}
	return &Migrator{fsys: fsys, opts: opt}, nil
}

// Up applies all unapplied migrations in ascending version order.
func (m *Migrator) Up(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("migrations: db is nil")
	}
	if err := m.ensureTable(ctx, db); err != nil {
		return err
	}
	migrations, err := loadMigrations(m.fsys, m.opts)
	if err != nil {
		return err
	}
	applied, err := m.appliedMap(ctx, db)
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		row, ok := applied[migration.Version]
		if ok {
			if row.Checksum != migration.Checksum {
				return fmt.Errorf("migrations: checksum mismatch for version %d", migration.Version)
			}
			continue
		}
		if err := m.applyUp(ctx, db, migration); err != nil {
			return err
		}
	}
	return nil
}

// Down rolls back the latest applied migrations. Steps must be positive.
func (m *Migrator) Down(ctx context.Context, db *sql.DB, steps int) error {
	if db == nil {
		return errors.New("migrations: db is nil")
	}
	if steps < 1 {
		return errors.New("migrations: down steps must be positive")
	}
	if err := m.ensureTable(ctx, db); err != nil {
		return err
	}
	local, err := loadMigrations(m.fsys, m.opts)
	if err != nil {
		return err
	}
	localByVersion := make(map[int64]Migration, len(local))
	for _, migration := range local {
		localByVersion[migration.Version] = migration
	}
	applied, err := m.appliedRows(ctx, db)
	if err != nil {
		return err
	}
	sort.Slice(applied, func(i, j int) bool {
		return applied[i].Version > applied[j].Version
	})
	if steps > len(applied) {
		steps = len(applied)
	}
	for i := 0; i < steps; i++ {
		row := applied[i]
		migration, ok := localByVersion[row.Version]
		if !ok {
			return fmt.Errorf("migrations: applied version %d has no local migration file", row.Version)
		}
		if migration.DownSQL == "" {
			return fmt.Errorf("migrations: version %d has no down migration", row.Version)
		}
		if err := m.applyDown(ctx, db, migration); err != nil {
			return err
		}
	}
	return nil
}

// Status returns the merged local-file and applied-database state.
func (m *Migrator) Status(ctx context.Context, db *sql.DB) ([]Status, error) {
	if db == nil {
		return nil, errors.New("migrations: db is nil")
	}
	if err := m.ensureTable(ctx, db); err != nil {
		return nil, err
	}
	local, err := loadMigrations(m.fsys, m.opts)
	if err != nil {
		return nil, err
	}
	applied, err := m.appliedMap(ctx, db)
	if err != nil {
		return nil, err
	}
	seen := make(map[int64]bool, len(local))
	statuses := make([]Status, 0, len(local)+len(applied))
	for _, migration := range local {
		seen[migration.Version] = true
		row, ok := applied[migration.Version]
		statuses = append(statuses, Status{
			Version:          migration.Version,
			Name:             migration.Name,
			Applied:          ok,
			AppliedAt:        appliedAt(row.AppliedAt),
			Checksum:         migration.Checksum,
			ChecksumMismatch: ok && row.Checksum != migration.Checksum,
		})
	}
	for version, row := range applied {
		if seen[version] {
			continue
		}
		statuses = append(statuses, Status{
			Version:   row.Version,
			Name:      row.Name,
			Applied:   true,
			AppliedAt: appliedAt(row.AppliedAt),
			Checksum:  row.Checksum,
			Missing:   true,
		})
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Version < statuses[j].Version
	})
	return statuses, nil
}

func (m *Migrator) applyUp(ctx context.Context, db *sql.DB, migration Migration) error {
	if m.opts.DisableTransactions {
		if _, err := db.ExecContext(ctx, migration.UpSQL); err != nil {
			return fmt.Errorf("migrations: apply version %d: %w", migration.Version, err)
		}
		return m.insertApplied(ctx, db, migration)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrations: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, migration.UpSQL); err != nil {
		return fmt.Errorf("migrations: apply version %d: %w", migration.Version, err)
	}
	if err := m.insertApplied(ctx, tx, migration); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrations: commit version %d: %w", migration.Version, err)
	}
	committed = true
	return nil
}

func (m *Migrator) applyDown(ctx context.Context, db *sql.DB, migration Migration) error {
	if m.opts.DisableTransactions {
		if _, err := db.ExecContext(ctx, migration.DownSQL); err != nil {
			return fmt.Errorf("migrations: rollback version %d: %w", migration.Version, err)
		}
		return m.deleteApplied(ctx, db, migration.Version)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrations: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, migration.DownSQL); err != nil {
		return fmt.Errorf("migrations: rollback version %d: %w", migration.Version, err)
	}
	if err := m.deleteApplied(ctx, tx, migration.Version); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrations: commit rollback %d: %w", migration.Version, err)
	}
	committed = true
	return nil
}

func (m *Migrator) ensureTable(ctx context.Context, db execer) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
version BIGINT PRIMARY KEY,
name VARCHAR(128) NOT NULL,
checksum VARCHAR(64) NOT NULL,
applied_at TIMESTAMP NOT NULL
)`, m.opts.TableName)
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("migrations: ensure schema table: %w", err)
	}
	return nil
}

func (m *Migrator) insertApplied(ctx context.Context, db execer, migration Migration) error {
	query := fmt.Sprintf("INSERT INTO %s (version, name, checksum, applied_at) VALUES (%s, %s, %s, %s)",
		m.opts.TableName,
		m.opts.Placeholder(1),
		m.opts.Placeholder(2),
		m.opts.Placeholder(3),
		m.opts.Placeholder(4),
	)
	_, err := db.ExecContext(ctx, query, migration.Version, migration.Name, migration.Checksum, m.opts.Now().UTC())
	if err != nil {
		return fmt.Errorf("migrations: record version %d: %w", migration.Version, err)
	}
	return nil
}

func (m *Migrator) deleteApplied(ctx context.Context, db execer, version int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE version = %s", m.opts.TableName, m.opts.Placeholder(1))
	_, err := db.ExecContext(ctx, query, version)
	if err != nil {
		return fmt.Errorf("migrations: delete version %d: %w", version, err)
	}
	return nil
}

func (m *Migrator) appliedMap(ctx context.Context, db *sql.DB) (map[int64]AppliedMigration, error) {
	rows, err := m.appliedRows(ctx, db)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]AppliedMigration, len(rows))
	for _, row := range rows {
		out[row.Version] = row
	}
	return out, nil
}

func (m *Migrator) appliedRows(ctx context.Context, db *sql.DB) ([]AppliedMigration, error) {
	query := fmt.Sprintf("SELECT version, name, checksum, applied_at FROM %s ORDER BY version ASC", m.opts.TableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("migrations: query applied versions: %w", err)
	}
	defer rows.Close()

	var out []AppliedMigration
	for rows.Next() {
		if len(out) >= m.opts.MaxMigrations {
			return nil, errors.New("migrations: too many applied migration rows")
		}
		var row AppliedMigration
		if err := rows.Scan(&row.Version, &row.Name, &row.Checksum, &row.AppliedAt); err != nil {
			return nil, fmt.Errorf("migrations: scan applied version: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrations: read applied versions: %w", err)
	}
	return out, nil
}

func appliedAt(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}
