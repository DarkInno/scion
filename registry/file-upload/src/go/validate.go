package fileupload

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrEmptyFile is returned when an uploaded file has no content.
	ErrEmptyFile = errors.New("fileupload: empty file")
	// ErrUnknownType is returned when the file type cannot be identified by magic bytes.
	ErrUnknownType = errors.New("fileupload: unknown file type")
	// ErrTypeNotAllowed is returned when the detected type is not in the whitelist.
	ErrTypeNotAllowed = errors.New("fileupload: file type not allowed")
)

// FileType describes a recognized file type.
type FileType struct {
	// MIME is the canonical MIME type.
	MIME string
	// Extension is the file extension including the leading dot, derived from the
	// detected type (never from the user-supplied filename).
	Extension string
	// Detect returns true if the leading bytes match this type.
	Detect func(b []byte) bool
}

// KnownTypes lists every file type recognized via magic bytes, in detection
// order. Add new types here to extend support.
var KnownTypes = []FileType{
	{MIME: "image/jpeg", Extension: ".jpg", Detect: IsJPEG},
	{MIME: "image/png", Extension: ".png", Detect: IsPNG},
	{MIME: "image/gif", Extension: ".gif", Detect: IsGIF},
	{MIME: "image/webp", Extension: ".webp", Detect: IsWebP},
	{MIME: "application/pdf", Extension: ".pdf", Detect: IsPDF},
}

// IsJPEG matches the JPEG SOI marker FF D8 FF.
func IsJPEG(b []byte) bool {
	return len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF
}

// IsPNG matches the PNG 8-byte signature.
func IsPNG(b []byte) bool {
	header := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	return len(b) >= 8 && bytes.Equal(b[:8], header)
}

// IsGIF matches the GIF87a or GIF89a header.
func IsGIF(b []byte) bool {
	if len(b) < 6 {
		return false
	}
	return bytes.Equal(b[:6], []byte("GIF87a")) || bytes.Equal(b[:6], []byte("GIF89a"))
}

// IsWebP matches the RIFF....WEBP header.
func IsWebP(b []byte) bool {
	if len(b) < 12 {
		return false
	}
	return bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP"))
}

// IsPDF matches the leading %PDF magic.
func IsPDF(b []byte) bool {
	return len(b) >= 4 && bytes.Equal(b[:4], []byte("%PDF"))
}

// DetectFileType identifies the FileType of data by inspecting its magic bytes.
// The boolean result is false if no known signature matches.
func DetectFileType(data []byte) (FileType, bool) {
	for _, t := range KnownTypes {
		if t.Detect(data) {
			return t, true
		}
	}
	return FileType{}, false
}

// IsAllowed reports whether mime is contained in the allowed whitelist
// (case-insensitive comparison).
func IsAllowed(mime string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(a, mime) {
			return true
		}
	}
	return false
}

// ValidateContent verifies that data is non-empty, has a recognized magic-byte
// signature, and that its type is permitted by the allowed whitelist.
func ValidateContent(data []byte, allowed []string) (FileType, error) {
	if len(data) == 0 {
		return FileType{}, ErrEmptyFile
	}
	ft, ok := DetectFileType(data)
	if !ok {
		return FileType{}, ErrUnknownType
	}
	if !IsAllowed(ft.MIME, allowed) {
		return FileType{}, ErrTypeNotAllowed
	}
	return ft, nil
}

// SanitizeFilename strips dangerous characters from a user-supplied filename:
// path separators, ".." traversal segments, NUL bytes, and CR/LF. The result is
// the base name only. An unsafe or empty input yields an empty string.
//
// The returned value is intended only for informational purposes (e.g. echoing
// back the original name); the actual stored name is always generated server-side.
func SanitizeFilename(name string) string {
	if name == "" {
		return ""
	}
	// Remove NUL and CRLF, which can break headers, logs, and filenames.
	name = strings.ReplaceAll(name, "\x00", "")
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\n", "")

	// Normalize Windows path separators before extracting base name.
	// filepath.Base on Linux does not recognize backslashes.
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)

	// Defense in depth: strip any remaining separators and dot-segments that
	// could survive on unusual platforms or inputs.
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, string(os.PathSeparator), "")
	name = strings.ReplaceAll(name, "..", "")

	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return ""
	}
	return name
}
