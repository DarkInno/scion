package fileupload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Shared test helpers (also used by pentest_test.go)
// Valid magic-byte payloads for every supported type.
func jpegBytes() []byte { return []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 16, 'J', 'F', 'I', 'F'} }
func pngBytes() []byte  { return []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 1, 2, 3} }
func gifBytes() []byte  { return []byte("GIF89a animated payload here") }
func pdfBytes() []byte  { return []byte("%PDF-1.7\n%binary\n") }

func webpBytes() []byte {
	b := make([]byte, 30)
	copy(b[0:4], []byte("RIFF"))
	// size bytes 4..7 can be anything
	copy(b[8:12], []byte("WEBP"))
	copy(b[12:16], []byte("VP8 "))
	return b
}

// exeBytes returns a payload with a Windows PE (MZ) header, an executable
// signature that must never be accepted as an upload.
func exeBytes() []byte { return []byte{'M', 'Z', 0x90, 0x00, 0x03, 0x00, 0x00, 0x00} }

// multipartBody builds a multipart/form-data body containing a single file part
// named field, with the given filename, declared Content-Type and content bytes.
// If declaredCT is empty, no Content-Type header is set on the part (the handler
// must not rely on it anyway). It returns the body and the full Content-Type
// header value (including the boundary) to use on the request.
func multipartBody(field, filename, declaredCT string, content []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field, filename))
	if declaredCT != "" {
		hdr.Set("Content-Type", declaredCT)
	}
	part, _ := w.CreatePart(hdr)
	_, _ = part.Write(content)
	_ = w.Close()

	return body, w.FormDataContentType()
}

// multipartTwoFiles builds a body with two file parts (used to test multi-file
// upload handling).
func multipartTwoFiles(field1, file1 string, ct1 string, c1 []byte,
	field2, file2 string, ct2 string, c2 []byte,
) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	h1 := textproto.MIMEHeader{}
	h1.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field1, file1))
	if ct1 != "" {
		h1.Set("Content-Type", ct1)
	}
	p1, _ := w.CreatePart(h1)
	_, _ = p1.Write(c1)

	h2 := textproto.MIMEHeader{}
	h2.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field2, file2))
	if ct2 != "" {
		h2.Set("Content-Type", ct2)
	}
	p2, _ := w.CreatePart(h2)
	_, _ = p2.Write(c2)

	_ = w.Close()
	return body, w.FormDataContentType()
}

func doUpload(h http.Handler, body *bytes.Buffer, ct string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", ct)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func decodeUpload(t *testing.T, rr *httptest.ResponseRecorder) UploadResponse {
	t.Helper()
	var resp UploadResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v (body=%q)", err, rr.Body.String())
	}
	return resp
}

func decodeError(t *testing.T, rr *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()
	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v (body=%q)", err, rr.Body.String())
	}
	return resp
}

// memoryHandler builds a Handler backed by MemoryStorage with rate limiting
// disabled, applying optional modifications to the options.
func memoryHandler(t *testing.T, modify func(*Options)) (*Handler, *MemoryStorage) {
	t.Helper()
	opts := Defaults()
	opts.Storage = NewMemoryStorage("/files")
	opts.RateLimit = 0
	if modify != nil {
		modify(opts)
	}
	h, err := NewHandler(opts)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h, opts.Storage.(*MemoryStorage)
}

func tmpLocalStorage(t *testing.T) (*Handler, *LocalStorage, string) {
	t.Helper()
	dir := t.TempDir()
	opts := Defaults()
	opts.UploadDir = dir
	opts.URLPrefix = "/files"
	opts.RateLimit = 0
	h, err := NewHandler(opts)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h, h.Storage().(*LocalStorage), dir
}

// config.go tests
func TestDefaults(t *testing.T) {
	o := Defaults()
	if o.MaxFileSize != DefaultMaxFileSize {
		t.Errorf("MaxFileSize = %d, want %d", o.MaxFileSize, DefaultMaxFileSize)
	}
	if o.RateLimit != DefaultRateLimit {
		t.Errorf("RateLimit = %d, want %d", o.RateLimit, DefaultRateLimit)
	}
	if o.RateWindow != DefaultRateWindow {
		t.Errorf("RateWindow = %v, want %v", o.RateWindow, DefaultRateWindow)
	}
	if o.FilenameFunc == nil {
		t.Fatal("FilenameFunc should be set by Defaults")
	}
	if o.Storage != nil {
		t.Fatal("Defaults should leave Storage nil so it has no FS side effects")
	}
	// Defaults must return an independent AllowedTypes slice.
	o.AllowedTypes[0] = "mutated"
	if DefaultAllowedTypes[0] == "mutated" {
		t.Fatal("Defaults did not copy the default AllowedTypes slice")
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("FILEUPLOAD_MAX_FILE_SIZE", "1234567")
	t.Setenv("FILEUPLOAD_RATE_LIMIT", "5")
	t.Setenv("FILEUPLOAD_RATE_WINDOW", "30s")
	t.Setenv("FILEUPLOAD_UPLOAD_DIR", "./up")
	t.Setenv("FILEUPLOAD_URL_PREFIX", "/u")
	t.Setenv("FILEUPLOAD_ALLOWED_TYPES", "image/png, application/pdf")

	o := FromEnv()
	if o.MaxFileSize != 1234567 {
		t.Errorf("MaxFileSize = %d, want 1234567", o.MaxFileSize)
	}
	if o.RateLimit != 5 {
		t.Errorf("RateLimit = %d, want 5", o.RateLimit)
	}
	if o.RateWindow != 30*time.Second {
		t.Errorf("RateWindow = %v, want 30s", o.RateWindow)
	}
	if o.UploadDir != "./up" {
		t.Errorf("UploadDir = %q, want ./up", o.UploadDir)
	}
	if o.URLPrefix != "/u" {
		t.Errorf("URLPrefix = %q, want /u", o.URLPrefix)
	}
	if len(o.AllowedTypes) != 2 || o.AllowedTypes[0] != "image/png" || o.AllowedTypes[1] != "application/pdf" {
		t.Errorf("AllowedTypes = %v, want [image/png application/pdf]", o.AllowedTypes)
	}
}

func TestFromEnvIgnoresGarbage(t *testing.T) {
	t.Setenv("FILEUPLOAD_MAX_FILE_SIZE", "not-a-number")
	t.Setenv("FILEUPLOAD_RATE_WINDOW", "bad")
	o := FromEnv()
	if o.MaxFileSize != DefaultMaxFileSize {
		t.Errorf("garbage MAX_FILE_SIZE should keep default, got %d", o.MaxFileSize)
	}
	if o.RateWindow != DefaultRateWindow {
		t.Errorf("garbage RATE_WINDOW should keep default, got %v", o.RateWindow)
	}
}

func TestGenerateUUIDv7(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		s, err := generateUUIDv7()
		if err != nil {
			t.Fatalf("generateUUIDv7: %v", err)
		}
		// 8-4-4-4-12 hex layout => 36 chars with dashes.
		if len(s) != 36 {
			t.Fatalf("uuid length = %d, want 36 (%q)", len(s), s)
		}
		// Version 7 => char at index 14 is '7'.
		if s[14] != '7' {
			t.Fatalf("uuid version = %c, want 7 (%q)", s[14], s)
		}
		// Variant 10 => first nibble of the 4th group is 8,9,a or b.
		switch s[19] {
		case '8', '9', 'a', 'b':
		default:
			t.Fatalf("uuid variant = %c, want 8/9/a/b (%q)", s[19], s)
		}
		if _, dup := seen[s]; dup {
			t.Fatalf("duplicate uuid generated: %s", s)
		}
		seen[s] = struct{}{}
	}
}

// validate.go tests
func TestDetectFileType(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		mime string
	}{
		{"jpeg", jpegBytes(), "image/jpeg"},
		{"png", pngBytes(), "image/png"},
		{"gif", gifBytes(), "image/gif"},
		{"webp", webpBytes(), "image/webp"},
		{"pdf", pdfBytes(), "application/pdf"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ft, ok := DetectFileType(c.data)
			if !ok {
				t.Fatalf("expected detection for %s", c.name)
			}
			if ft.MIME != c.mime {
				t.Errorf("MIME = %q, want %q", ft.MIME, c.mime)
			}
			if ft.Extension == "" {
				t.Errorf("Extension should be non-empty for %s", c.name)
			}
		})
	}
}

func TestDetectFileTypeRejectsUnknown(t *testing.T) {
	if _, ok := DetectFileType(exeBytes()); ok {
		t.Fatal("MZ/PE executable must not be detected as a known type")
	}
	if _, ok := DetectFileType(nil); ok {
		t.Fatal("nil data must not be detected")
	}
	if _, ok := DetectFileType([]byte{0x00, 0x01, 0x02}); ok {
		t.Fatal("random bytes must not be detected")
	}
}

func TestValidateContent(t *testing.T) {
	allowed := []string{"image/jpeg", "application/pdf"}

	if _, err := ValidateContent(nil, allowed); err != ErrEmptyFile {
		t.Errorf("empty file: err = %v, want ErrEmptyFile", err)
	}
	if _, err := ValidateContent(exeBytes(), allowed); err != ErrUnknownType {
		t.Errorf("unknown type: err = %v, want ErrUnknownType", err)
	}
	// gif is known but not in the allow-list.
	if _, err := ValidateContent(gifBytes(), allowed); err != ErrTypeNotAllowed {
		t.Errorf("disallowed type: err = %v, want ErrTypeNotAllowed", err)
	}
	ft, err := ValidateContent(jpegBytes(), allowed)
	if err != nil {
		t.Fatalf("jpeg should be valid: %v", err)
	}
	if ft.MIME != "image/jpeg" {
		t.Errorf("MIME = %q, want image/jpeg", ft.MIME)
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"photo.jpg", "photo.jpg"},
		{"../../etc/passwd", "passwd"},
		{"/etc/passwd", "passwd"},
		{"..\\..\\windows\\system32\\cmd.exe", "cmd.exe"},
		{"a\x00b.jpg", "ab.jpg"},
		{"a\r\nb.png", "ab.png"},
		{"..%2f..%2fsecret", "%2f%2fsecret"}, // % chars are not separators; base keeps them
		{"", ""},
		{"...", ""},
		{".", ""},
		{"   ", ""},
		{"normal file name.jpeg", "normal file name.jpeg"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := SanitizeFilename(c.in)
			if got != c.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", c.in, got, c.want)
			}
			// The sanitized result must never contain a separator or traversal.
			if strings.Contains(got, "..") || strings.ContainsAny(got, `/\`) {
				t.Errorf("sanitized result %q still contains traversal/separator", got)
			}
		})
	}
}

// storage.go tests
func TestMemoryStorageCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage("/files")

	url, err := s.Save(ctx, "abc.jpg", jpegBytes())
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if url != "/files/abc.jpg" {
		t.Errorf("url = %q, want /files/abc.jpg", url)
	}
	if !s.Exists(ctx, "abc.jpg") {
		t.Fatal("Exists should report true after Save")
	}
	got, err := s.Get(ctx, "abc.jpg")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, jpegBytes()) {
		t.Fatal("Get returned different bytes than saved")
	}
	// Get must return a copy: mutating it must not affect storage.
	got[0] = 0x00
	again, _ := s.Get(ctx, "abc.jpg")
	if again[0] != 0xFF {
		t.Fatal("Get did not return an independent copy")
	}
	if err := s.Delete(ctx, "abc.jpg"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s.Exists(ctx, "abc.jpg") {
		t.Fatal("Exists should report false after Delete")
	}
	if _, err := s.Get(ctx, "abc.jpg"); err != ErrFileNotFound {
		t.Errorf("Get after delete: err = %v, want ErrFileNotFound", err)
	}
	// Delete of a missing file is not an error.
	if err := s.Delete(ctx, "missing"); err != nil {
		t.Errorf("Delete missing: %v", err)
	}
}

func TestMemoryStorageRejectsBadNames(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage("/files")
	bad := []string{
		"../escape.jpg",
		"a/b.jpg",
		"a\\b.jpg",
		"..\\..\\x",
		"a\x00b.png",
		"a\nb.gif",
		"",
	}
	for _, name := range bad {
		if _, err := s.Save(ctx, name, jpegBytes()); err != ErrInvalidName {
			t.Errorf("Save(%q): err = %v, want ErrInvalidName", name, err)
		}
		if s.Exists(ctx, name) {
			t.Errorf("Exists(%q) should be false for a rejected name", name)
		}
	}
}

func TestLocalStorageCRUD(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := NewLocalStorage(dir, "/files")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	// RootDir should be absolute.
	if !filepath.IsAbs(s.RootDir) {
		t.Errorf("RootDir = %q, expected absolute", s.RootDir)
	}
	url, err := s.Save(ctx, "abc.png", pngBytes())
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if url != "/files/abc.png" {
		t.Errorf("url = %q, want /files/abc.png", url)
	}
	if !s.Exists(ctx, "abc.png") {
		t.Fatal("Exists should be true")
	}
	got, err := s.Get(ctx, "abc.png")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, pngBytes()) {
		t.Fatal("Get returned different bytes")
	}
	// The file must actually exist on disk at RootDir/name (no subdirectories).
	if _, err := os.Stat(filepath.Join(s.RootDir, "abc.png")); err != nil {
		t.Fatalf("file not on disk where expected: %v", err)
	}
	if err := s.Delete(ctx, "abc.png"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "abc.png"); err != ErrFileNotFound {
		t.Errorf("Get after delete: err = %v, want ErrFileNotFound", err)
	}
}

func TestLocalStorageRejectsTraversal(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	// Put a sentinel file outside the root that a traversal would overwrite.
	parent := filepath.Dir(dir)
	sentinel := filepath.Join(parent, "sentinel.txt")
	_ = os.WriteFile(sentinel, []byte("keep"), 0o644)

	s, err := NewLocalStorage(dir, "/files")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	bad := []string{
		"../sentinel.txt",
		"..\\sentinel.txt",
		"sub/../../sentinel.txt",
		"a/../sentinel.txt",
		"./sentinel.txt",
	}
	for _, name := range bad {
		if _, err := s.Save(ctx, name, jpegBytes()); err != ErrInvalidName {
			t.Errorf("Save(%q): err = %v, want ErrInvalidName", name, err)
		}
	}
	// Ensure the sentinel was never overwritten/created by the store.
	if data, err := os.ReadFile(sentinel); err == nil {
		if string(data) != "keep" {
			t.Fatal("sentinel file was modified despite traversal protection")
		}
	}
}

// handler.go tests
func TestUploadEachType(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		mime string
	}{
		{"jpeg", jpegBytes(), "image/jpeg"},
		{"png", pngBytes(), "image/png"},
		{"gif", gifBytes(), "image/gif"},
		{"webp", webpBytes(), "image/webp"},
		{"pdf", pdfBytes(), "application/pdf"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, store := memoryHandler(t, nil)
			body, ct := multipartBody("file", "upload.bin", "", c.data)
			rr := doUpload(h, body, ct)

			if rr.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d (body=%s)", rr.Code, http.StatusCreated, rr.Body.String())
			}
			if got := rr.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("response Content-Type = %q, want application/json", got)
			}
			resp := decodeUpload(t, rr)
			if !resp.Success {
				t.Fatal("response Success should be true")
			}
			if resp.File.MimeType != c.mime {
				t.Errorf("MimeType = %q, want %q", resp.File.MimeType, c.mime)
			}
			if resp.File.Size != int64(len(c.data)) {
				t.Errorf("Size = %d, want %d", resp.File.Size, len(c.data))
			}
			if !strings.HasPrefix(resp.File.URL, "/files/") {
				t.Errorf("URL = %q, want /files/ prefix", resp.File.URL)
			}
			if resp.File.Name == "" {
				t.Fatal("Name should not be empty")
			}
			// The stored name must be the generated one with the right extension.
			wantExt := mimeToExt(c.mime)
			if !strings.HasSuffix(resp.File.Name, wantExt) {
				t.Errorf("Name = %q, want suffix %q", resp.File.Name, wantExt)
			}
			// The stored name must contain no path separators or traversal.
			if strings.ContainsAny(resp.File.Name, `/\`) || strings.Contains(resp.File.Name, "..") {
				t.Errorf("stored Name %q contains traversal/separator", resp.File.Name)
			}
			// Original name should be sanitized (we used upload.bin).
			if resp.File.OriginalName != "upload.bin" {
				t.Errorf("OriginalName = %q, want upload.bin", resp.File.OriginalName)
			}
			// The bytes must be persisted under the generated name.
			saved, err := store.Get(nil, resp.File.Name)
			if err != nil {
				t.Fatalf("storage Get: %v", err)
			}
			if !bytes.Equal(saved, c.data) {
				t.Fatal("stored bytes differ from uploaded bytes")
			}
		})
	}
}

func mimeToExt(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	}
	return ""
}

func TestUploadLocalStorage(t *testing.T) {
	h, store, _ := tmpLocalStorage(t)
	body, ct := multipartBody("file", "img.png", "", pngBytes())
	rr := doUpload(h, body, ct)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", rr.Code, rr.Body.String())
	}
	resp := decodeUpload(t, rr)
	if !store.Exists(nil, resp.File.Name) {
		t.Fatal("file should exist on local disk after upload")
	}
}

func TestUploadRejectsEmptyFile(t *testing.T) {
	h, _ := memoryHandler(t, nil)
	body, ct := multipartBody("file", "empty.jpg", "image/jpeg", []byte{})
	rr := doUpload(h, body, ct)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	er := decodeError(t, rr)
	if !strings.Contains(er.Error, "empty") {
		t.Errorf("error = %q, want mention 'empty'", er.Error)
	}
}

func TestUploadRejectsUnknownType(t *testing.T) {
	h, _ := memoryHandler(t, nil)
	// Declared as image/jpeg but content is an executable; magic bytes win.
	body, ct := multipartBody("file", "trojan.exe", "image/jpeg", exeBytes())
	rr := doUpload(h, body, ct)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	er := decodeError(t, rr)
	if !strings.Contains(er.Error, "unknown") {
		t.Errorf("error = %q, want mention 'unknown'", er.Error)
	}
}

func TestUploadRejectsDisallowedType(t *testing.T) {
	h, _ := memoryHandler(t, func(o *Options) {
		o.AllowedTypes = []string{"application/pdf"} // only PDF allowed
	})
	// gif is a known type but not in the allow-list.
	body, ct := multipartBody("file", "img.gif", "image/gif", gifBytes())
	rr := doUpload(h, body, ct)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415", rr.Code)
	}
	er := decodeError(t, rr)
	if !strings.Contains(er.Error, "not allowed") {
		t.Errorf("error = %q, want mention 'not allowed'", er.Error)
	}
}

func TestUploadIgnoresContentTypeHeader(t *testing.T) {
	// Declared as application/pdf but content is JPEG: must be accepted as JPEG.
	h, _ := memoryHandler(t, nil)
	body, ct := multipartBody("file", "lie.pdf", "application/pdf", jpegBytes())
	rr := doUpload(h, body, ct)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", rr.Code, rr.Body.String())
	}
	resp := decodeUpload(t, rr)
	if resp.File.MimeType != "image/jpeg" {
		t.Errorf("MimeType = %q, want image/jpeg (magic bytes must override header)", resp.File.MimeType)
	}
	if !strings.HasSuffix(resp.File.Name, ".jpg") {
		t.Errorf("Name = %q, want .jpg suffix from detected type", resp.File.Name)
	}
}

func TestUploadOversizedFile(t *testing.T) {
	h, _ := memoryHandler(t, func(o *Options) {
		o.MaxFileSize = 64
	})
	big := bytes.Repeat([]byte{0xFF, 0xD8, 0xFF}, 100) // 300 bytes > 64
	body, ct := multipartBody("file", "big.jpg", "image/jpeg", big)
	rr := doUpload(h, body, ct)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rr.Code)
	}
}

func TestUploadMethodNotAllowed(t *testing.T) {
	h, _ := memoryHandler(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/upload", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestUploadNoFilePart(t *testing.T) {
	h, _ := memoryHandler(t, nil)
	// A multipart body with only a normal text field, no file.
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	_ = w.WriteField("username", "alice")
	_ = w.Close()
	rr := doUpload(h, body, w.FormDataContentType())
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	er := decodeError(t, rr)
	if !strings.Contains(er.Error, "no file") {
		t.Errorf("error = %q, want mention 'no file'", er.Error)
	}
}

func TestRateLimit(t *testing.T) {
	opts := Defaults()
	opts.Storage = NewMemoryStorage("/files")
	opts.RateLimit = 2
	opts.RateWindow = time.Minute
	h, err := NewHandler(opts)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	upload := func() int {
		body, ct := multipartBody("file", "a.jpg", "image/jpeg", jpegBytes())
		return doUpload(h, body, ct).Code
	}

	if code := upload(); code != http.StatusCreated {
		t.Fatalf("1st upload: code = %d, want 201", code)
	}
	if code := upload(); code != http.StatusCreated {
		t.Fatalf("2nd upload: code = %d, want 201", code)
	}
	if code := upload(); code != http.StatusTooManyRequests {
		t.Fatalf("3rd upload: code = %d, want 429", code)
	}
}

func TestRateLimitByRemoteAddr(t *testing.T) {
	opts := Defaults()
	opts.Storage = NewMemoryStorage("/files")
	opts.RateLimit = 1
	opts.RateWindow = time.Minute
	h, err := NewHandler(opts)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	doWithRemoteAddr := func(addr string) int {
		body, ct := multipartBody("file", "a.jpg", "image/jpeg", jpegBytes())
		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", ct)
		req.RemoteAddr = addr
		// XFF must NOT affect rate limiting (security fix).
		req.Header.Set("X-Forwarded-For", "10.0.0.99")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr.Code
	}
	if code := doWithRemoteAddr("10.0.0.1:12345"); code != http.StatusCreated {
		t.Fatalf("first IP 1st: code = %d, want 201", code)
	}
	if code := doWithRemoteAddr("10.0.0.1:12345"); code != http.StatusTooManyRequests {
		t.Fatalf("first IP 2nd: code = %d, want 429", code)
	}
	// A different RemoteAddr has its own bucket.
	if code := doWithRemoteAddr("10.0.0.2:12345"); code != http.StatusCreated {
		t.Fatalf("second IP 1st: code = %d, want 201", code)
	}
	// XFF spoofing must NOT bypass rate limiting: same RemoteAddr, different XFF.
	if code := doWithRemoteAddr("10.0.0.2:12345"); code != http.StatusTooManyRequests {
		t.Fatalf("XFF spoofing bypassed rate limit: code = %d, want 429", code)
	}
}

func TestMiddlewareInterceptsAndPassesThrough(t *testing.T) {
	// Use in-memory storage so the test does not create ./uploads on disk.
	mwOpts := Defaults()
	mwOpts.Storage = NewMemoryStorage("/files")
	mwOpts.RateLimit = 0
	mw := Middleware(mwOpts)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})
	wrapped := mw(next)

	// Non-multipart GET must pass through to next.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if !called {
		t.Fatal("next handler was not called for non-multipart request")
	}
	if rr.Code != http.StatusTeapot {
		t.Errorf("passthrough code = %d, want 418", rr.Code)
	}

	// Multipart POST must be handled by the upload handler, not next.
	called = false
	body, ct := multipartBody("file", "a.jpg", "image/jpeg", jpegBytes())
	req2 := httptest.NewRequest(http.MethodPost, "/upload", body)
	req2.Header.Set("Content-Type", ct)
	rr2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr2, req2)
	if called {
		t.Fatal("next handler should not be called for a multipart upload")
	}
	if rr2.Code != http.StatusCreated {
		t.Errorf("upload code = %d, want 201", rr2.Code)
	}
}

func TestNewHandlerAppliesDefaults(t *testing.T) {
	// Provide only the storage location; every other zero field must be filled
	// in with safe defaults so the source tree is not polluted with ./uploads.
	opts := &Options{
		UploadDir: t.TempDir(),
		URLPrefix: "/files",
	}
	h, err := NewHandler(opts)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	if h.Options().MaxFileSize != DefaultMaxFileSize {
		t.Errorf("MaxFileSize = %d, want default", h.Options().MaxFileSize)
	}
	if len(h.Options().AllowedTypes) != len(DefaultAllowedTypes) {
		t.Errorf("AllowedTypes len = %d, want %d", len(h.Options().AllowedTypes), len(DefaultAllowedTypes))
	}
	if h.Options().FilenameFunc == nil {
		t.Error("FilenameFunc should default to a generator")
	}
	if h.Options().RateWindow != DefaultRateWindow {
		t.Errorf("RateWindow = %v, want default", h.Options().RateWindow)
	}
	if h.Storage() == nil {
		t.Fatal("Storage should be created when nil")
	}
	if _, ok := h.Storage().(*LocalStorage); !ok {
		t.Fatalf("Storage should be LocalStorage, got %T", h.Storage())
	}
	// FromEnv must also yield a working handler without polluting the source tree.
	t.Setenv("FILEUPLOAD_UPLOAD_DIR", t.TempDir())
	if _, err := NewHandler(FromEnv()); err != nil {
		t.Fatalf("NewHandler(FromEnv): %v", err)
	}
}

func TestGeneratedNamesAreUnique(t *testing.T) {
	h, store := memoryHandler(t, nil)
	seen := make(map[string]struct{}, 50)
	for i := 0; i < 50; i++ {
		body, ct := multipartBody("file", "x.jpg", "image/jpeg", jpegBytes())
		rr := doUpload(h, body, ct)
		if rr.Code != http.StatusCreated {
			t.Fatalf("upload %d: code = %d", i, rr.Code)
		}
		resp := decodeUpload(t, rr)
		if _, dup := seen[resp.File.Name]; dup {
			t.Fatalf("duplicate generated name: %s", resp.File.Name)
		}
		seen[resp.File.Name] = struct{}{}
	}
	if len(seen) != 50 {
		t.Fatalf("expected 50 distinct files, got %d", len(seen))
	}
	// Sanity: ensure no stored name contains anything but [hex-] + .jpg.
	for name := range store.files {
		if !strings.HasSuffix(name, ".jpg") {
			t.Errorf("name %q missing .jpg suffix", name)
		}
	}
}
