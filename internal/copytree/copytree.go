package copytree

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type File struct {
	RelPath string
	Data    []byte
	Mode    os.FileMode
}

type Options struct {
	DryRun bool
	Force  bool
}

type Result struct {
	Files []string
}

func CopyFiles(root string, files []File, opts Options) (Result, error) {
	if strings.TrimSpace(root) == "" {
		return Result{}, fmt.Errorf("target directory is required")
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	rootAbs = filepath.Clean(rootAbs)

	targets := make([]struct {
		file File
		path string
	}, 0, len(files))
	for _, file := range files {
		targetPath, err := SafeJoin(rootAbs, file.RelPath)
		if err != nil {
			return Result{}, err
		}
		if info, err := os.Stat(targetPath); err == nil {
			if info.IsDir() {
				return Result{}, fmt.Errorf("target path is a directory: %s", file.RelPath)
			}
			if !opts.Force {
				return Result{}, fmt.Errorf("target file already exists: %s", file.RelPath)
			}
		} else if !os.IsNotExist(err) {
			return Result{}, err
		}
		targets = append(targets, struct {
			file File
			path string
		}{file: file, path: targetPath})
	}

	var copied []string
	for _, target := range targets {
		copied = append(copied, normalizeRel(target.file.RelPath))
		if opts.DryRun {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target.path), 0o755); err != nil {
			return Result{}, err
		}
		mode := target.file.Mode
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(target.path, target.file.Data, mode); err != nil {
			return Result{}, err
		}
	}

	return Result{Files: copied}, nil
}

func SafeJoin(rootAbs, rel string) (string, error) {
	cleanRel, err := CleanRelativePath(rel)
	if err != nil {
		return "", err
	}

	target := filepath.Join(rootAbs, filepath.FromSlash(cleanRel))
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	targetAbs = filepath.Clean(targetAbs)

	inside, err := isInside(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if !inside {
		return "", fmt.Errorf("target escapes destination root: %s", rel)
	}
	return targetAbs, nil
}

func CleanRelativePath(rel string) (string, error) {
	if strings.ContainsAny(rel, "\x00\r\n") {
		return "", fmt.Errorf("path contains control characters")
	}
	if strings.Contains(rel, "\\") {
		return "", fmt.Errorf("path must use slash separators")
	}
	if path.IsAbs(rel) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute path is not allowed")
	}
	clean := path.Clean(rel)
	if clean == "." || clean == "" {
		return "", fmt.Errorf("empty path")
	}
	for _, part := range strings.Split(clean, "/") {
		if part == ".." || part == "" {
			return "", fmt.Errorf("path traversal is not allowed")
		}
	}
	return clean, nil
}

func isInside(rootAbs, targetAbs string) (bool, error) {
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false, err
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))), nil
}

func normalizeRel(rel string) string {
	clean, err := CleanRelativePath(rel)
	if err != nil {
		return rel
	}
	return clean
}
