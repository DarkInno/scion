package registry_test

import (
	"testing"

	"github.com/DarkInno/scion/internal/bundle"
	"github.com/DarkInno/scion/internal/registry"
)

func TestEmbeddedBundleLoadsSchemaV1Index(t *testing.T) {
	reg, err := registry.NewBundle(bundle.RegistryZip, bundle.ManifestJSON)
	if err != nil {
		t.Fatalf("load embedded bundle: %v", err)
	}
	if reg.Index.SchemaVersion != registry.IndexSchemaVersion {
		t.Fatalf("schema = %d", reg.Index.SchemaVersion)
	}
	if _, ok := reg.Index.Module("cache"); !ok {
		t.Fatal("cache module not found")
	}
	if reg.Manifest.BundleHash == "" {
		t.Fatal("bundle hash is empty")
	}
}

func TestModuleFilesExcludeGoModUnlessStandalone(t *testing.T) {
	reg, err := registry.NewBundle(bundle.RegistryZip, bundle.ManifestJSON)
	if err != nil {
		t.Fatalf("load embedded bundle: %v", err)
	}
	module, ok := reg.Index.Module("cache")
	if !ok {
		t.Fatal("cache module not found")
	}

	files, err := reg.ModuleFiles(module, false)
	if err != nil {
		t.Fatalf("module files: %v", err)
	}
	for _, file := range files {
		if file.RelativePath == "go.mod" {
			t.Fatal("go.mod should not be copied by default")
		}
	}

	files, err = reg.ModuleFiles(module, true)
	if err != nil {
		t.Fatalf("standalone module files: %v", err)
	}
	foundGoMod := false
	for _, file := range files {
		if file.RelativePath == "go.mod" {
			foundGoMod = true
		}
	}
	if !foundGoMod {
		t.Fatal("go.mod should be copied in standalone mode")
	}
}

func TestCleanBundlePathRejectsZipSlip(t *testing.T) {
	for _, path := range []string{"../x", "/x", "a\\b", "a/\x00b"} {
		if _, err := registry.CleanBundlePath(path); err == nil {
			t.Fatalf("expected %q to be rejected", path)
		}
	}
}
