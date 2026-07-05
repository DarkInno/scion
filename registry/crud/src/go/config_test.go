package crud

import (
	"os"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	os.Setenv("DB_URL", "postgres://localhost/test")
	defer os.Unsetenv("DB_URL")
	defer os.Unsetenv("DEFAULT_PAGE_SIZE")
	defer os.Unsetenv("MAX_PAGE_SIZE")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DBURL != "postgres://localhost/test" {
		t.Errorf("expected DBURL, got %s", cfg.DBURL)
	}
	if cfg.DefaultPageSize != 20 {
		t.Errorf("expected default page size 20, got %d", cfg.DefaultPageSize)
	}
	if cfg.MaxPageSize != 100 {
		t.Errorf("expected default max page size 100, got %d", cfg.MaxPageSize)
	}
}

func TestLoadConfig_MissingDBURL(t *testing.T) {
	os.Unsetenv("DB_URL")
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing DB_URL")
	}
}

func TestLoadConfig_CustomSizes(t *testing.T) {
	os.Setenv("DB_URL", "postgres://localhost/test")
	os.Setenv("DEFAULT_PAGE_SIZE", "50")
	os.Setenv("MAX_PAGE_SIZE", "200")
	defer os.Unsetenv("DB_URL")
	defer os.Unsetenv("DEFAULT_PAGE_SIZE")
	defer os.Unsetenv("MAX_PAGE_SIZE")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DefaultPageSize != 50 {
		t.Errorf("expected default page size 50, got %d", cfg.DefaultPageSize)
	}
	if cfg.MaxPageSize != 200 {
		t.Errorf("expected max page size 200, got %d", cfg.MaxPageSize)
	}
}

func TestLoadConfig_DefaultExceedsMax(t *testing.T) {
	os.Setenv("DB_URL", "postgres://localhost/test")
	os.Setenv("DEFAULT_PAGE_SIZE", "150")
	os.Setenv("MAX_PAGE_SIZE", "100")
	defer os.Unsetenv("DB_URL")
	defer os.Unsetenv("DEFAULT_PAGE_SIZE")
	defer os.Unsetenv("MAX_PAGE_SIZE")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DefaultPageSize != 100 {
		t.Errorf("expected default page size capped to max 100, got %d", cfg.DefaultPageSize)
	}
}

func TestDefaultSortValidator(t *testing.T) {
	if DefaultSortValidator("anything") {
		t.Error("DefaultSortValidator should reject all fields")
	}
	if DefaultSortValidator("") {
		t.Error("DefaultSortValidator should reject empty field")
	}
}
