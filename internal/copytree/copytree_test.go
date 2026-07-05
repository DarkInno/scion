package copytree

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCleanRelativePathRejectsUnsafePaths(t *testing.T) {
	tests := []string{
		"",
		".",
		"../secret",
		"safe/../../secret",
		"safe\\file.go",
		"safe/\x00file.go",
		"safe/\r\nfile.go",
	}
	if runtime.GOOS == "windows" {
		tests = append(tests, `C:/secret/file.go`)
	} else {
		tests = append(tests, `/secret/file.go`)
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			if _, err := CleanRelativePath(test); err == nil {
				t.Fatalf("expected %q to be rejected", test)
			}
		})
	}
}

func TestCopyFilesRefusesOverwriteUnlessForced(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cache.go"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := CopyFiles(dir, []File{{RelPath: "cache.go", Data: []byte("new")}}, Options{})
	if err == nil {
		t.Fatal("expected overwrite without force to fail")
	}

	if _, err := CopyFiles(dir, []File{{RelPath: "cache.go", Data: []byte("new")}}, Options{Force: true}); err != nil {
		t.Fatalf("forced copy failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "cache.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("expected forced copy to replace file, got %q", string(data))
	}
}

func TestSafeJoinRejectsEscape(t *testing.T) {
	root, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SafeJoin(root, "../outside.go"); err == nil {
		t.Fatal("expected path escape to fail")
	}
	if got, err := SafeJoin(root, "internal/cache/cache.go"); err != nil {
		t.Fatalf("safe path failed: %v", err)
	} else if filepath.Dir(got) != filepath.Join(root, "internal", "cache") {
		t.Fatalf("unexpected joined path: %s", got)
	}
}
