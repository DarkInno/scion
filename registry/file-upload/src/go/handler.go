package fileupload

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// FileInfo describes a successfully uploaded file.
type FileInfo struct {
	URL          string `json:"url"`
	Name         string `json:"name"`
	OriginalName string `json:"originalName"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mimeType"`
}

// UploadResponse is the JSON body returned for a successful upload.
type UploadResponse struct {
	Success bool     `json:"success"`
	File    FileInfo `json:"file"`
}

// ErrorResponse is the JSON body returned for a failed upload.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// Handler handles multipart/form-data file uploads. It is safe for concurrent
// use and implements http.Handler.
type Handler struct {
	opts    *Options
	storage Storage
	limiter *rateLimiter
}

// NewHandler builds a Handler from opts. If opts is nil, Defaults() is used.
// If opts.Storage is nil, a LocalStorage is created from opts.UploadDir and
// opts.URLPrefix. A non-zero RateLimit enables per-client throttling.
func NewHandler(opts *Options) (*Handler, error) {
	if opts == nil {
		opts = Defaults()
	}
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = DefaultMaxFileSize
	}
	if len(opts.AllowedTypes) == 0 {
		opts.AllowedTypes = append([]string(nil), DefaultAllowedTypes...)
	}
	if opts.UploadDir == "" {
		opts.UploadDir = DefaultUploadDir
	}
	if opts.URLPrefix == "" {
		opts.URLPrefix = DefaultURLPrefix
	}
	if opts.FilenameFunc == nil {
		opts.FilenameFunc = generateUUIDv7
	}
	if opts.RateWindow <= 0 {
		opts.RateWindow = DefaultRateWindow
	}

	h := &Handler{opts: opts}
	if opts.Storage != nil {
		h.storage = opts.Storage
	} else {
		ls, err := NewLocalStorage(opts.UploadDir, opts.URLPrefix)
		if err != nil {
			return nil, err
		}
		h.storage = ls
	}
	if opts.RateLimit > 0 {
		h.limiter = newRateLimiter(opts.RateLimit, opts.RateWindow)
	}
	return h, nil
}

// Storage returns the backing storage (useful for inspection in tests).
func (h *Handler) Storage() Storage { return h.storage }

// Options returns the resolved options.
func (h *Handler) Options() *Options { return h.opts }

// ServeHTTP implements http.Handler. It accepts a single file from a
// multipart/form-data request (any part carrying a filename) and stores it.
//
// On success it responds with HTTP 201 and a JSON body containing the file URL,
// generated name, sanitized original name, size, and detected MIME type.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.limiter != nil {
		if !h.limiter.Allow(clientIP(r)) {
			writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
	}
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed: use POST")
		return
	}

	// Cap the entire request body so a client cannot stream unlimited bytes. The
	// extra 1 MiB covers multipart framing and any non-file form fields.
	maxBody := h.opts.MaxFileSize + 1<<20
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)

	reader, err := r.MultipartReader()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid multipart request: "+err.Error())
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "failed to read multipart part: "+err.Error())
			return
		}

		// Ignore non-file form fields; drain them so the reader can advance.
		if part.FileName() == "" {
			_, _ = io.Copy(io.Discard, part)
			continue
		}

		// Read at most MaxFileSize+1 bytes so an oversized file is detected
		// without buffering the entire upload into memory.
		limited := io.LimitReader(part, h.opts.MaxFileSize+1)
		data, err := io.ReadAll(limited)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "failed to read file content: "+err.Error())
			return
		}
		if int64(len(data)) > h.opts.MaxFileSize {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "file exceeds maximum allowed size")
			return
		}

		// Verify content by magic bytes; the Content-Type header is never trusted.
		ft, err := ValidateContent(data, h.opts.AllowedTypes)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrTypeNotAllowed) {
				status = http.StatusUnsupportedMediaType
			}
			writeJSONError(w, status, err.Error())
			return
		}

		// Generate a fresh, unpredictable base name and append the extension
		// derived from the detected type (never the client-supplied extension).
		baseName, err := h.opts.FilenameFunc()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to generate filename: "+err.Error())
			return
		}
		name := SanitizeFilename(baseName) + ft.Extension
		if name == ft.Extension || name == "" {
			// The generated base name was unsafe; refuse to store.
			writeJSONError(w, http.StatusInternalServerError, "generated filename is invalid")
			return
		}

		ctx := r.Context()
		url, err := h.storage.Save(ctx, name, data)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to store file: "+err.Error())
			return
		}

		resp := UploadResponse{
			Success: true,
			File: FileInfo{
				URL:          url,
				Name:         name,
				OriginalName: SanitizeFilename(part.FileName()),
				Size:         int64(len(data)),
				MimeType:     ft.MIME,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	writeJSONError(w, http.StatusBadRequest, "no file part found in request")
}

// Middleware returns an http middleware with the signature
// func(http.Handler) http.Handler. It intercepts multipart POST uploads and
// stores them, passing all other requests through to next.
//
// This makes the module usable both as a dedicated upload endpoint and as a
// composable middleware in an existing router.
func Middleware(opts *Options) func(http.Handler) http.Handler {
	h, err := NewHandler(opts)
	if err != nil {
		panic(err)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && isMultipart(r.Header.Get("Content-Type")) {
				h.ServeHTTP(w, r)
				return
			}
			if next != nil {
				next.ServeHTTP(w, r)
			}
		})
	}
}

func isMultipart(ct string) bool {
	return strings.HasPrefix(strings.ToLower(ct), "multipart/form-data")
}

// clientIP extracts the remote client address from the TCP connection.
// SECURITY: X-Forwarded-For is NOT trusted — it is client-controlled and
// can be spoofed to bypass rate limiting. If you are behind a trusted reverse
// proxy, use the middleware module's TrustedProxy middleware instead.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// writeJSONError writes a JSON error response. The encoder error is intentionally
// ignored: once the status code and headers are written there is no useful
// recovery action, and the client either receives the body or a truncated stream.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Success: false, Error: msg})
}

// rateLimiter is a simple per-key fixed-window limiter. It is process-local and
// sufficient for protecting a single instance from upload abuse.
type rateLimiter struct {
	mu     sync.Mutex
	state  map[string]*bucket
	rate   int
	window time.Duration
}

// maxBuckets limits the number of tracked keys to prevent memory exhaustion.
const maxBuckets = 10000

type bucket struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		state:  make(map[string]*bucket),
		rate:   rate,
		window: window,
	}
}

// Allow reports whether key may perform one more action within the current window.
func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()

	// Periodic cleanup: remove expired buckets to prevent memory growth.
	if len(rl.state) >= maxBuckets {
		for k, b := range rl.state {
			if now.After(b.resetAt) {
				delete(rl.state, k)
			}
		}
		// If still at capacity, evict the bucket with the earliest reset time.
		if len(rl.state) >= maxBuckets {
			var oldestKey string
			var oldestTime time.Time
			first := true
			for k, b := range rl.state {
				if first || b.resetAt.Before(oldestTime) {
					oldestKey = k
					oldestTime = b.resetAt
					first = false
				}
			}
			delete(rl.state, oldestKey)
		}
	}

	b, ok := rl.state[key]
	if !ok || now.After(b.resetAt) {
		rl.state[key] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.rate {
		return false
	}
	b.count++
	return true
}
