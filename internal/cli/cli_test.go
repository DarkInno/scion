package cli_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DarkInno/scion/internal/bundle"
	"github.com/DarkInno/scion/internal/cli"
	"github.com/DarkInno/scion/internal/registry"
)

func newTestApp(t *testing.T) *cli.App {
	t.Helper()
	reg, err := registry.NewBundle(bundle.RegistryZip, bundle.ManifestJSON)
	if err != nil {
		t.Fatalf("load embedded bundle: %v", err)
	}
	return cli.New(reg)
}

func TestAddCacheAndDiffInTemporaryGoProject(t *testing.T) {
	app := newTestApp(t)
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/app\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(project, "internal", "cache")

	var stdout, stderr bytes.Buffer
	code := app.Run(context.Background(), []string{"add", "cache", "--to", target}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("add exit %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(target, ".scion-module.json")); err != nil {
		t.Fatalf("metadata missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "go.mod")); !os.IsNotExist(err) {
		t.Fatalf("go.mod should not be copied by default, err=%v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = project
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("copied cache package does not test cleanly: %v\n%s", err, string(out))
	}

	stdout.Reset()
	stderr.Reset()
	code = app.Run(context.Background(), []string{"diff", "cache", "--target", target}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("diff exit %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	if err := os.WriteFile(filepath.Join(target, "local.txt"), []byte("user file"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = app.Run(context.Background(), []string{"diff", "cache", "--target", target}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("diff with added file exit %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "local.txt") {
		t.Fatalf("diff output did not mention added file: %q", stdout.String())
	}
}

func TestAddRefusesExternalDependencyModuleWithoutStandalone(t *testing.T) {
	app := newTestApp(t)
	var stdout, stderr bytes.Buffer
	code := app.Run(context.Background(), []string{"add", "auth", "--to", filepath.Join(t.TempDir(), "internal", "auth")}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("add auth exit %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "stdlibOnly=false") {
		t.Fatalf("expected stdlibOnly error, got %q", stderr.String())
	}
}

func TestListJSON(t *testing.T) {
	app := newTestApp(t)
	var stdout, stderr bytes.Buffer
	code := app.Run(context.Background(), []string{"list", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("list exit %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id":"cache"`) {
		t.Fatalf("list json missing cache: %q", stdout.String())
	}
}
