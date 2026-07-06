// Package migrations provides a small, standard-library-only database migration
// runner for copy-paste Go services.
package migrations

import (
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// Placeholder returns a SQL placeholder for the 1-based argument position.
type Placeholder func(position int) string

// Options configures migration discovery and SQL bookkeeping.
type Options struct {
	// Dir is the directory inside the provided fs.FS that contains migration
	// files. Defaults to ".".
	Dir string
	// TableName is the trusted schema history table name. Dot-qualified names
	// such as "public.schema_migrations" are allowed.
	TableName string
	// MaxMigrations caps the number of migration files loaded from Dir.
	MaxMigrations int
	// MaxSQLBytes caps the size of each migration SQL file.
	MaxSQLBytes int64
	// DisableTransactions executes migration SQL and bookkeeping outside an
	// explicit database transaction. Keep false unless your driver or migration
	// statement cannot run in a transaction.
	DisableTransactions bool
	// Placeholder controls bind marker syntax. Use DollarPlaceholder for
	// PostgreSQL-style drivers.
	Placeholder Placeholder
	// Now supplies timestamps for schema history rows. Defaults to time.Now.
	Now func() time.Time
}

// Defaults returns secure defaults for local SQL migration files.
func Defaults() Options {
	return Options{
		Dir:           ".",
		TableName:     "schema_migrations",
		MaxMigrations: 512,
		MaxSQLBytes:   1 << 20,
		Placeholder:   QuestionPlaceholder,
		Now:           time.Now,
	}
}

// FromEnv reads migration options from environment variables.
//
// Supported variables:
//   - MIGRATIONS_DIR
//   - MIGRATIONS_TABLE
//   - MIGRATIONS_MAX_FILES
//   - MIGRATIONS_MAX_SQL_BYTES
//   - MIGRATIONS_DISABLE_TRANSACTIONS
//   - MIGRATIONS_PLACEHOLDER_STYLE ("question" or "dollar")
func FromEnv() Options {
	o := Defaults()
	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		o.Dir = v
	}
	if v := os.Getenv("MIGRATIONS_TABLE"); v != "" {
		o.TableName = v
	}
	if v := os.Getenv("MIGRATIONS_MAX_FILES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			o.MaxMigrations = n
		}
	}
	if v := os.Getenv("MIGRATIONS_MAX_SQL_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			o.MaxSQLBytes = n
		}
	}
	if v := os.Getenv("MIGRATIONS_DISABLE_TRANSACTIONS"); v != "" {
		o.DisableTransactions = strings.EqualFold(v, "true")
	}
	if v := strings.ToLower(os.Getenv("MIGRATIONS_PLACEHOLDER_STYLE")); v == "dollar" {
		o.Placeholder = DollarPlaceholder
	}
	return o
}

func (o Options) normalize() (Options, error) {
	d := Defaults()
	if o.Dir == "" {
		o.Dir = d.Dir
	}
	if o.TableName == "" {
		o.TableName = d.TableName
	}
	if o.MaxMigrations <= 0 {
		o.MaxMigrations = d.MaxMigrations
	}
	if o.MaxMigrations > 10000 {
		o.MaxMigrations = 10000
	}
	if o.MaxSQLBytes <= 0 {
		o.MaxSQLBytes = d.MaxSQLBytes
	}
	if o.MaxSQLBytes > 64<<20 {
		o.MaxSQLBytes = 64 << 20
	}
	if o.Placeholder == nil {
		o.Placeholder = d.Placeholder
	}
	if o.Now == nil {
		o.Now = d.Now
	}
	if err := validateDir(o.Dir); err != nil {
		return Options{}, err
	}
	o.Dir = path.Clean(o.Dir)
	if err := validateTableName(o.TableName); err != nil {
		return Options{}, err
	}
	return o, nil
}

// QuestionPlaceholder returns "?" for MySQL, SQLite, and other drivers that
// use anonymous placeholders.
func QuestionPlaceholder(_ int) string {
	return "?"
}

// DollarPlaceholder returns PostgreSQL-style placeholders: $1, $2, ...
func DollarPlaceholder(position int) string {
	if position < 1 {
		position = 1
	}
	return "$" + strconv.Itoa(position)
}
