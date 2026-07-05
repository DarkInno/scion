package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DarkInno/scion/internal/registry"
)

var fixedZipTime = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

func main() {
	root := flag.String("root", ".", "repository root")
	outDir := flag.String("out", "internal/bundle", "bundle output directory")
	flag.Parse()

	if err := run(*root, *outDir); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "build-bundle: %v\n", err)
		os.Exit(1)
	}
}

func run(root, outDir string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	indexBytes, err := os.ReadFile(filepath.Join(rootAbs, "registry", "index.json"))
	if err != nil {
		return err
	}
	idx, err := registry.ParseIndex(indexBytes)
	if err != nil {
		return err
	}

	paths, err := collectRegistryFiles(rootAbs)
	if err != nil {
		return err
	}

	manifest := registry.Manifest{
		SchemaVersion:   1,
		RegistryVersion: idx.Version,
		Modules:         make([]registry.BundleModule, 0, len(idx.Patterns)),
		Files:           make([]registry.BundleFile, 0, len(paths)),
	}
	for _, module := range idx.SortedModules() {
		manifest.Modules = append(manifest.Modules, registry.BundleModule{
			ID:      module.ID,
			Version: module.Version,
			Source:  module.Source,
		})
	}

	moduleByPath := make([]registry.Module, 0, len(idx.Patterns))
	moduleByPath = append(moduleByPath, idx.Patterns...)
	sort.Slice(moduleByPath, func(i, j int) bool {
		return len(moduleByPath[i].Path) > len(moduleByPath[j].Path)
	})

	var zipData bytes.Buffer
	zipWriter := zip.NewWriter(&zipData)
	for _, rel := range paths {
		abs := filepath.Join(rootAbs, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		header := &zip.FileHeader{
			Name:     rel,
			Method:   zip.Store,
			Modified: fixedZipTime,
		}
		header.SetMode(0o644)
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}

		sum := sha256.Sum256(data)
		entry := registry.BundleFile{
			Path:   rel,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   int64(len(data)),
		}
		if module, ok := moduleForPath(moduleByPath, rel); ok {
			entry.Module = module.ID
			entry.ModuleVersion = module.Version
		}
		manifest.Files = append(manifest.Files, entry)
	}
	if err := zipWriter.Close(); err != nil {
		return err
	}

	bundleHash := sha256.Sum256(zipData.Bytes())
	manifest.BundleHash = hex.EncodeToString(bundleHash[:])
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	manifestBytes = append(manifestBytes, '\n')

	outAbs, err := filepath.Abs(filepath.Join(rootAbs, outDir))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outAbs, "registry.zip"), zipData.Bytes(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outAbs, "manifest.json"), manifestBytes, 0o644); err != nil {
		return err
	}
	return nil
}

func collectRegistryFiles(root string) ([]string, error) {
	registryRoot := filepath.Join(root, "registry")
	var paths []string
	if err := filepath.WalkDir(registryRoot, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if _, err := registry.CleanBundlePath(rel); err != nil {
			return err
		}
		paths = append(paths, rel)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func moduleForPath(modules []registry.Module, rel string) (registry.Module, bool) {
	for _, module := range modules {
		prefix := strings.TrimSuffix(module.Path, "/") + "/"
		if rel == module.Path || strings.HasPrefix(rel, prefix) {
			return module, true
		}
	}
	return registry.Module{}, false
}
