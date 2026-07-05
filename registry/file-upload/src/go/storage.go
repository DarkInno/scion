package fileupload

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	// ErrFileNotFound is returned when a stored file does not exist.
	ErrFileNotFound = errors.New("fileupload: file not found")
	// ErrInvalidName is returned when a file name contains path separators,
	// traversal segments, or control characters and is therefore rejected by the
	// storage layer.
	ErrInvalidName = errors.New("fileupload: invalid filename")
)

// Storage abstracts where and how uploaded files are persisted. Implementations
// must reject any name that contains path separators or traversal segments so
// that path-traversal attacks cannot escape the storage root.
type Storage interface {
	// Save stores data under name and returns the public URL of the file.
	Save(ctx context.Context, name string, data []byte) (url string, err error)
	// Get retrieves the bytes of a stored file.
	Get(ctx context.Context, name string) (data []byte, err error)
	// Delete removes a stored file. A missing file is not an error.
	Delete(ctx context.Context, name string) error
	// Exists reports whether a file with the given name is currently stored.
	Exists(ctx context.Context, name string) bool
}

// safeName validates that name is a bare file name: non-empty and free of path
// separators, traversal segments, and control characters. It is the single
// chokepoint that prevents path traversal at the storage boundary.
func safeName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	if strings.ContainsAny(name, `/\`) {
		return ErrInvalidName
	}
	if strings.Contains(name, "..") {
		return ErrInvalidName
	}
	if strings.ContainsAny(name, "\x00\r\n") {
		return ErrInvalidName
	}
	// filepath.Base should be a no-op for an already-clean name; if it changes,
	// the name contained a separator on this platform and must be rejected.
	if filepath.Base(name) != name {
		return ErrInvalidName
	}
	return nil
}

// LocalStorage persists files on the local disk under a single root directory.
type LocalStorage struct {
	RootDir   string
	URLPrefix string
}

// NewLocalStorage creates a LocalStorage rooted at rootDir, creating the
// directory (and parents) if it does not yet exist.
func NewLocalStorage(rootDir, urlPrefix string) (*LocalStorage, error) {
	abs, err := filepath.Abs(filepath.Clean(rootDir))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	return &LocalStorage{RootDir: abs, URLPrefix: urlPrefix}, nil
}

// resolve validates name and returns the absolute on-disk path, ensuring the
// resolved path stays within RootDir (defense in depth beyond safeName).
func (s *LocalStorage) resolve(name string) (string, error) {
	if err := safeName(name); err != nil {
		return "", err
	}
	full := filepath.Join(s.RootDir, name)
	rel, err := filepath.Rel(s.RootDir, full)
	if err != nil {
		return "", ErrInvalidName
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", ErrInvalidName
	}
	return full, nil
}

// Save implements Storage.
func (s *LocalStorage) Save(ctx context.Context, name string, data []byte) (string, error) {
	full, err := s.resolve(name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	return strings.TrimRight(s.URLPrefix, "/") + "/" + name, nil
}

// Get implements Storage.
func (s *LocalStorage) Get(ctx context.Context, name string) ([]byte, error) {
	full, err := s.resolve(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	return data, nil
}

// Delete implements Storage.
func (s *LocalStorage) Delete(ctx context.Context, name string) error {
	full, err := s.resolve(name)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Exists implements Storage.
func (s *LocalStorage) Exists(ctx context.Context, name string) bool {
	full, err := s.resolve(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(full)
	return err == nil
}

// MemoryStorage keeps files in memory. It is mainly useful for tests, ephemeral
// services, and deployments where persistence is handled elsewhere.
type MemoryStorage struct {
	URLPrefix string

	mu    sync.RWMutex
	files map[string][]byte
}

// NewMemoryStorage creates an empty MemoryStorage.
func NewMemoryStorage(urlPrefix string) *MemoryStorage {
	return &MemoryStorage{
		URLPrefix: urlPrefix,
		files:     make(map[string][]byte),
	}
}

// Save implements Storage.
func (s *MemoryStorage) Save(ctx context.Context, name string, data []byte) (string, error) {
	if err := safeName(name); err != nil {
		return "", err
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	s.mu.Lock()
	s.files[name] = cp
	s.mu.Unlock()
	return strings.TrimRight(s.URLPrefix, "/") + "/" + name, nil
}

// Get implements Storage.
func (s *MemoryStorage) Get(ctx context.Context, name string) ([]byte, error) {
	s.mu.RLock()
	data, ok := s.files[name]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrFileNotFound
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

// Delete implements Storage.
func (s *MemoryStorage) Delete(ctx context.Context, name string) error {
	s.mu.Lock()
	delete(s.files, name)
	s.mu.Unlock()
	return nil
}

// Exists implements Storage.
func (s *MemoryStorage) Exists(ctx context.Context, name string) bool {
	s.mu.RLock()
	_, ok := s.files[name]
	s.mu.RUnlock()
	return ok
}
