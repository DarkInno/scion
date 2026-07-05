package registry

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

type Bundle struct {
	Index    *Index
	Manifest Manifest

	files        map[string]*zip.File
	manifestFile map[string]BundleFile
}

type ModuleFile struct {
	SourcePath   string
	RelativePath string
	SHA256       string
	Size         int64
}

func NewBundle(zipBytes, manifestBytes []byte) (*Bundle, error) {
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported bundle manifest schema %d", manifest.SchemaVersion)
	}

	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("open registry bundle: %w", err)
	}

	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		clean, err := CleanBundlePath(file.Name)
		if err != nil {
			return nil, fmt.Errorf("unsafe bundle path %q: %w", file.Name, err)
		}
		files[clean] = file
	}

	indexBytes, err := readZipFile(files, "registry/index.json")
	if err != nil {
		return nil, err
	}
	idx, err := ParseIndex(indexBytes)
	if err != nil {
		return nil, fmt.Errorf("parse registry index: %w", err)
	}

	manifestFile := make(map[string]BundleFile, len(manifest.Files))
	for _, file := range manifest.Files {
		manifestFile[file.Path] = file
	}

	hash := sha256.Sum256(zipBytes)
	if got := hex.EncodeToString(hash[:]); manifest.BundleHash != "" && got != manifest.BundleHash {
		return nil, fmt.Errorf("bundle hash mismatch: manifest has %s, zip has %s", manifest.BundleHash, got)
	}

	return &Bundle{
		Index:        idx,
		Manifest:     manifest,
		files:        files,
		manifestFile: manifestFile,
	}, nil
}

func (b *Bundle) ReadFile(name string) ([]byte, error) {
	clean, err := CleanBundlePath(name)
	if err != nil {
		return nil, err
	}
	return readZipFile(b.files, clean)
}

func (b *Bundle) HasFile(name string) bool {
	clean, err := CleanBundlePath(name)
	if err != nil {
		return false
	}
	_, ok := b.files[clean]
	return ok
}

func (b *Bundle) ListFiles(prefix string) []string {
	prefix = strings.TrimSuffix(path.Clean(prefix), "/")
	if prefix == "." {
		prefix = ""
	}
	if prefix != "" {
		prefix += "/"
	}

	var out []string
	for name := range b.files {
		if strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func (b *Bundle) ModuleFiles(module Module, standalone bool) ([]ModuleFile, error) {
	source, err := CleanBundlePath(module.Source)
	if err != nil {
		return nil, fmt.Errorf("invalid source for %s: %w", module.ID, err)
	}
	sourcePrefix := strings.TrimSuffix(source, "/") + "/"

	patterns := append([]string(nil), module.Include...)
	if standalone {
		patterns = append(patterns, module.StandaloneInclude...)
	}
	if len(patterns) == 0 {
		patterns = []string{"*.go"}
	}

	var out []ModuleFile
	for _, name := range b.ListFiles(sourcePrefix) {
		file := b.files[name]
		if file.FileInfo().IsDir() {
			continue
		}
		rel := strings.TrimPrefix(name, sourcePrefix)
		if rel == "" || strings.Contains(rel, "/") && !matchesAny(patterns, rel) {
			continue
		}
		if !matchesAny(patterns, rel) {
			continue
		}

		entry, ok := b.manifestFile[name]
		if !ok {
			return nil, fmt.Errorf("bundle manifest missing %s", name)
		}
		out = append(out, ModuleFile{
			SourcePath:   name,
			RelativePath: rel,
			SHA256:       entry.SHA256,
			Size:         entry.Size,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].RelativePath < out[j].RelativePath
	})
	return out, nil
}

func CleanBundlePath(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty path")
	}
	if strings.ContainsAny(name, "\x00\r\n") {
		return "", fmt.Errorf("path contains control characters")
	}
	if strings.Contains(name, "\\") {
		return "", fmt.Errorf("path must use slash separators")
	}
	if path.IsAbs(name) {
		return "", fmt.Errorf("absolute path is not allowed")
	}
	clean := path.Clean(name)
	if clean == "." || clean == "" {
		return "", fmt.Errorf("empty path")
	}
	for _, part := range strings.Split(clean, "/") {
		if part == ".." {
			return "", fmt.Errorf("path traversal is not allowed")
		}
	}
	return clean, nil
}

func readZipFile(files map[string]*zip.File, name string) ([]byte, error) {
	file, ok := files[name]
	if !ok {
		return nil, fmt.Errorf("bundle file not found: %s", name)
	}
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

func matchesAny(patterns []string, rel string) bool {
	rel = path.Clean(rel)
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		cleanPattern := path.Clean(pattern)
		if cleanPattern == rel {
			return true
		}
		ok, err := path.Match(cleanPattern, rel)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func CanonicalFileBytes(name string, data []byte) []byte {
	if !isTextBundlePath(name) || !bytes.Contains(data, []byte{'\r'}) {
		return data
	}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	return data
}

func isTextBundlePath(name string) bool {
	base := strings.ToLower(path.Base(name))
	switch base {
	case ".gitignore", "dockerfile", "license", "makefile":
		return true
	}

	switch strings.ToLower(path.Ext(name)) {
	case ".css", ".env", ".go", ".html", ".js", ".json", ".jsx", ".md",
		".mod", ".sql", ".sum", ".tmpl", ".tpl", ".ts", ".tsx", ".txt",
		".yaml", ".yml":
		return true
	default:
		return false
	}
}
