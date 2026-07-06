package migrations

import (
	"testing"
	"testing/fstest"
)

func TestLoadMigrationsPairsAndSorts(t *testing.T) {
	fsys := fstest.MapFS{
		"20260101000002_add_posts.up.sql":   {Data: []byte("CREATE TABLE posts(id BIGINT);")},
		"20260101000002_add_posts.down.sql": {Data: []byte("DROP TABLE posts;")},
		"20260101000001_add_users.up.sql":   {Data: []byte("CREATE TABLE users(id BIGINT);")},
	}
	migrations, err := Load(fsys)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(migrations) != 2 {
		t.Fatalf("len = %d", len(migrations))
	}
	if migrations[0].Name != "add_users" || migrations[1].Name != "add_posts" {
		t.Fatalf("unexpected order: %+v", migrations)
	}
	if migrations[1].DownSQL == "" || migrations[0].Checksum == "" {
		t.Fatalf("missing migration fields: %+v", migrations)
	}
}

func TestLoadRejectsMissingUp(t *testing.T) {
	fsys := fstest.MapFS{
		"20260101000001_add_users.down.sql": {Data: []byte("DROP TABLE users;")},
	}
	if _, err := Load(fsys); err == nil {
		t.Fatalf("expected missing up error")
	}
}

func TestLoadRejectsInvalidFilename(t *testing.T) {
	fsys := fstest.MapFS{
		"20260101000001_bad.name.up.sql": {Data: []byte("SELECT 1;")},
	}
	if _, err := Load(fsys); err == nil {
		t.Fatalf("expected invalid filename error")
	}
}

func TestLoadRejectsLargeSQL(t *testing.T) {
	fsys := fstest.MapFS{
		"20260101000001_add_users.up.sql": {Data: []byte("SELECT 12345;")},
	}
	_, err := Load(fsys, Options{MaxSQLBytes: 4})
	if err == nil {
		t.Fatalf("expected max SQL size error")
	}
}
