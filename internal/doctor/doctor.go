package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DarkInno/scion/internal/registry"
)

type Issue struct {
	Level   string `json:"level"`
	Module  string `json:"module,omitempty"`
	Message string `json:"message"`
}

type Report struct {
	Strict bool    `json:"strict"`
	OK     bool    `json:"ok"`
	Issues []Issue `json:"issues,omitempty"`
}

func Run(reg *registry.Bundle, root string, strict bool) Report {
	var issues []Issue
	add := func(level, module, message string) {
		issues = append(issues, Issue{Level: level, Module: module, Message: message})
	}
	gated := func(module, message string) {
		if strict {
			add("error", module, message)
		} else {
			add("warning", module, message)
		}
	}

	if reg.Index.SchemaVersion != registry.IndexSchemaVersion {
		add("error", "", "registry index schema is unsupported")
	}
	if reg.Index.Version == "" {
		add("error", "", "registry version is empty")
	}

	seen := make(map[string]bool)
	for _, module := range reg.Index.SortedModules() {
		if module.ID == "" {
			add("error", "", "module id is empty")
			continue
		}
		if seen[module.ID] {
			add("error", module.ID, "duplicate module id")
		}
		seen[module.ID] = true

		if module.Path == "" || module.Source == "" {
			add("error", module.ID, "module path/source is required")
			continue
		}
		if len(reg.ListFiles(module.Path+"/")) == 0 {
			add("error", module.ID, "module path does not exist in bundle")
		}
		if len(reg.ListFiles(module.Source+"/")) == 0 {
			add("error", module.ID, "module source does not exist in bundle")
		}
		if module.Version == "" {
			add("error", module.ID, "module version is empty")
		}
		if module.Package == "" {
			add("error", module.ID, "module package is empty")
		}
		if module.DefaultTarget == "" {
			add("error", module.ID, "module defaultTarget is empty")
		}
		if len(module.Include) == 0 {
			add("error", module.ID, "module include list is empty")
		}
		if !reg.HasFile(module.Path + "/README.md") {
			gated(module.ID, "README.md is missing")
		}
		if !reg.HasFile(module.Path + "/__llms__.md") {
			gated(module.ID, "__llms__.md is missing")
		}
		if !reg.HasFile(module.Source + "/pentest_test.go") {
			add("error", module.ID, "pentest_test.go is missing")
		}
		if hasExternalRequire(reg, module) && module.StdlibOnly {
			gated(module.ID, "go.mod contains external dependencies but module is marked stdlibOnly")
		}
		if strict {
			checkTestPairing(reg, module, add)
		}
	}

	checkManifestFreshness(reg, root, add)

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Level != issues[j].Level {
			return issues[i].Level < issues[j].Level
		}
		if issues[i].Module != issues[j].Module {
			return issues[i].Module < issues[j].Module
		}
		return issues[i].Message < issues[j].Message
	})

	ok := true
	for _, issue := range issues {
		if issue.Level == "error" {
			ok = false
			break
		}
	}
	return Report{Strict: strict, OK: ok, Issues: issues}
}

func hasExternalRequire(reg *registry.Bundle, module registry.Module) bool {
	data, err := reg.ReadFile(module.Source + "/go.mod")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "require (" || strings.HasPrefix(line, "require ") {
			return true
		}
	}
	return false
}

func checkTestPairing(reg *registry.Bundle, module registry.Module, add func(string, string, string)) {
	files := reg.ListFiles(module.Source + "/")
	sourceFiles := make(map[string]bool)
	testFiles := make(map[string]bool)
	for _, file := range files {
		if !strings.HasSuffix(file, ".go") {
			continue
		}
		base := filepath.Base(file)
		if strings.HasSuffix(base, "_test.go") {
			testFiles[strings.TrimSuffix(base, "_test.go")] = true
			continue
		}
		sourceFiles[strings.TrimSuffix(base, ".go")] = true
	}
	for base := range sourceFiles {
		if !testFiles[base] {
			add("error", module.ID, base+".go has no corresponding _test.go")
		}
	}
}

func checkManifestFreshness(reg *registry.Bundle, root string, add func(string, string, string)) {
	if root == "" {
		return
	}
	registryRoot := filepath.Join(root, "registry")
	if _, err := os.Stat(registryRoot); err != nil {
		return
	}

	manifestFiles := make(map[string]registry.BundleFile)
	for _, file := range reg.Manifest.Files {
		if strings.HasPrefix(file.Path, "registry/") {
			manifestFiles[file.Path] = file
		}
	}
	seen := make(map[string]bool)
	for path, file := range manifestFiles {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			add("error", file.Module, "bundle is stale: missing "+path)
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			add("error", file.Module, "bundle is stale: hash mismatch for "+path)
		}
		seen[path] = true
	}

	_ = filepath.WalkDir(registryRoot, func(name string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !seen[rel] {
			add("error", "", "bundle is stale: untracked "+rel)
		}
		return nil
	})

	zipPath := filepath.Join(root, "internal", "bundle", "registry.zip")
	if data, err := os.ReadFile(zipPath); err == nil {
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != reg.Manifest.BundleHash {
			add("error", "", "bundle zip hash does not match manifest")
		}
	}
}
