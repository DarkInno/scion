package fileupload

import (
	"strings"
	"testing"
	"time"
)

func TestConfigDefaultsAndFromEnv(t *testing.T) {
	defaults := Defaults()
	if defaults.MaxFileSize != DefaultMaxFileSize {
		t.Fatalf("MaxFileSize = %d", defaults.MaxFileSize)
	}
	if defaults.FilenameFunc == nil {
		t.Fatal("FilenameFunc should be set")
	}

	t.Setenv("FILEUPLOAD_MAX_FILE_SIZE", "1024")
	t.Setenv("FILEUPLOAD_RATE_LIMIT", "7")
	t.Setenv("FILEUPLOAD_RATE_WINDOW", "2m")
	t.Setenv("FILEUPLOAD_UPLOAD_DIR", "/tmp/uploads")
	t.Setenv("FILEUPLOAD_URL_PREFIX", "/assets")
	t.Setenv("FILEUPLOAD_ALLOWED_TYPES", "image/png, application/pdf")

	opts := FromEnv()
	if opts.MaxFileSize != 1024 || opts.RateLimit != 7 || opts.RateWindow != 2*time.Minute {
		t.Fatalf("env options not applied: %+v", opts)
	}
	if opts.UploadDir != "/tmp/uploads" || opts.URLPrefix != "/assets" {
		t.Fatalf("env paths not applied: %+v", opts)
	}
	if got := strings.Join(opts.AllowedTypes, ","); got != "image/png,application/pdf" {
		t.Fatalf("allowed types = %q", got)
	}
}

func TestConfigGeneratedNameIsUUIDLike(t *testing.T) {
	name, err := generateUUIDv7()
	if err != nil {
		t.Fatalf("generateUUIDv7: %v", err)
	}
	if len(name) != 36 || name[14] != '7' {
		t.Fatalf("unexpected UUIDv7: %q", name)
	}
}
