package migrations

import (
	"testing"
	"time"
)

func TestDefaultsNormalize(t *testing.T) {
	opts, err := (Options{}).normalize()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if opts.Dir != "." || opts.TableName != "schema_migrations" {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
	if opts.Placeholder(2) != "?" {
		t.Fatalf("unexpected placeholder")
	}
	if opts.Now == nil || opts.Now().IsZero() {
		t.Fatalf("missing Now")
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("MIGRATIONS_DIR", "db/migrations")
	t.Setenv("MIGRATIONS_TABLE", "public.schema_migrations")
	t.Setenv("MIGRATIONS_MAX_FILES", "10")
	t.Setenv("MIGRATIONS_MAX_SQL_BYTES", "2048")
	t.Setenv("MIGRATIONS_DISABLE_TRANSACTIONS", "true")
	t.Setenv("MIGRATIONS_PLACEHOLDER_STYLE", "dollar")

	opts, err := FromEnv().normalize()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if opts.Dir != "db/migrations" || opts.TableName != "public.schema_migrations" {
		t.Fatalf("unexpected env options: %+v", opts)
	}
	if opts.MaxMigrations != 10 || opts.MaxSQLBytes != 2048 || !opts.DisableTransactions {
		t.Fatalf("unexpected limits: %+v", opts)
	}
	if got := opts.Placeholder(3); got != "$3" {
		t.Fatalf("placeholder = %q", got)
	}
}

func TestNormalizeRejectsUnsafeTable(t *testing.T) {
	_, err := Options{TableName: "schema_migrations;DROP", Now: time.Now}.normalize()
	if err == nil {
		t.Fatalf("expected unsafe table error")
	}
}

func TestNormalizeCleansSafeDirectory(t *testing.T) {
	opts, err := Options{Dir: "db/./migrations"}.normalize()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if opts.Dir != "db/migrations" {
		t.Fatalf("dir = %q", opts.Dir)
	}
}
