package fileupload

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultMaxFileSize is the default per-file size limit (10 MiB).
	DefaultMaxFileSize int64 = 10 * 1024 * 1024
	// DefaultRateLimit is the default number of uploads allowed per client per window.
	DefaultRateLimit = 60
	// DefaultRateWindow is the default rate-limit window.
	DefaultRateWindow = time.Minute
	// DefaultUploadDir is the default on-disk storage directory.
	DefaultUploadDir = "./uploads"
	// DefaultURLPrefix is the default URL prefix exposed for stored files.
	DefaultURLPrefix = "/files"
)

// DefaultAllowedTypes is the default whitelist of MIME types accepted via magic
// bytes. Extensions are never trusted.
var DefaultAllowedTypes = []string{
	"image/jpeg",
	"image/png",
	"image/gif",
	"image/webp",
	"application/pdf",
}

// Options configures the upload handler. Use Defaults() or FromEnv() to obtain a
// populated instance, then override individual fields as needed.
type Options struct {
	// MaxFileSize is the maximum allowed size of a single uploaded file, in bytes.
	MaxFileSize int64
	// AllowedTypes is the whitelist of MIME types verified via magic bytes.
	AllowedTypes []string
	// Storage is the backing storage. If nil, NewHandler creates a LocalStorage
	// from UploadDir/URLPrefix.
	Storage Storage
	// RateLimit is the maximum uploads per client per RateWindow. 0 disables limiting.
	RateLimit int
	// RateWindow is the duration over which RateLimit is counted.
	RateWindow time.Duration
	// UploadDir is the local directory used by the default LocalStorage.
	UploadDir string
	// URLPrefix is the URL prefix prepended to stored file names when building URLs.
	URLPrefix string
	// FilenameFunc generates a new, safe base name (without extension) for each
	// upload. It must not contain separators or traversal segments.
	FilenameFunc func() (string, error)
}

// Defaults returns a new Options populated with safe default values.
//
// Storage is intentionally left nil; NewHandler will lazily build a LocalStorage
// from UploadDir so that Defaults() itself has no filesystem side effects.
func Defaults() *Options {
	return &Options{
		MaxFileSize:  DefaultMaxFileSize,
		AllowedTypes: append([]string(nil), DefaultAllowedTypes...),
		RateLimit:    DefaultRateLimit,
		RateWindow:   DefaultRateWindow,
		UploadDir:    DefaultUploadDir,
		URLPrefix:    DefaultURLPrefix,
		FilenameFunc: generateUUIDv7,
	}
}

// FromEnv returns Defaults() overridden by environment variables:
//
//	FILEUPLOAD_MAX_FILE_SIZE  (bytes)
//	FILEUPLOAD_RATE_LIMIT     (uploads per window, 0 disables)
//	FILEUPLOAD_RATE_WINDOW    (Go duration, e.g. "30s")
//	FILEUPLOAD_UPLOAD_DIR     (filesystem path)
//	FILEUPLOAD_URL_PREFIX     (URL prefix, e.g. "/files")
//	FILEUPLOAD_ALLOWED_TYPES  (comma-separated MIME list)
//
// Invalid values are silently ignored so a misconfigured env var cannot break
// startup; the corresponding default is retained instead.
func FromEnv() *Options {
	opts := Defaults()

	if v := os.Getenv("FILEUPLOAD_MAX_FILE_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			opts.MaxFileSize = n
		}
	}
	if v := os.Getenv("FILEUPLOAD_RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			opts.RateLimit = n
		}
	}
	if v := os.Getenv("FILEUPLOAD_RATE_WINDOW"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			opts.RateWindow = d
		}
	}
	if v := os.Getenv("FILEUPLOAD_UPLOAD_DIR"); v != "" {
		opts.UploadDir = v
	}
	if v := os.Getenv("FILEUPLOAD_URL_PREFIX"); v != "" {
		opts.URLPrefix = v
	}
	if v := os.Getenv("FILEUPLOAD_ALLOWED_TYPES"); v != "" {
		parts := strings.Split(v, ",")
		cleaned := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				cleaned = append(cleaned, t)
			}
		}
		if len(cleaned) > 0 {
			opts.AllowedTypes = cleaned
		}
	}
	return opts
}

// generateUUIDv7 produces a time-ordered, cryptographically-random UUIDv7 string
// using crypto/rand. The timestamp prefix makes generated names naturally sorted
// by creation time, which is convenient for on-disk storage.
func generateUUIDv7() (string, error) {
	var u [16]byte

	// 48 bits of Unix milliseconds (big-endian).
	now := time.Now().UnixMilli()
	u[0] = byte(now >> 40)
	u[1] = byte(now >> 32)
	u[2] = byte(now >> 24)
	u[3] = byte(now >> 16)
	u[4] = byte(now >> 8)
	u[5] = byte(now)

	// 74 bits of randomness from a CSPRNG.
	if _, err := io.ReadFull(rand.Reader, u[6:]); err != nil {
		return "", err
	}

	// Set version 7.
	u[6] = (u[6] & 0x0F) | 0x70
	// Set variant 10 (RFC 4122).
	u[8] = (u[8] & 0x3F) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:]), nil
}
