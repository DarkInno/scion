package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// uuidv7Pattern matches the standard UUIDv7 format: 8-4-4-4-12 hex characters.
// Format: xxxxxxxx-xxxx-7xxx-8xxx-xxxxxxxxxxxx
// Version (7) is in the first nibble of group 3.
// Variant (8/9/a/b) is in the first nibble of group 4.
var uuidv7Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestRequestIDGenerated(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	mw := RequestID()
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected X-Request-ID header to be set, got empty")
	}

	if !uuidv7Pattern.MatchString(id) {
		t.Fatalf("generated ID %q does not match UUIDv7 format", id)
	}

	// Verify the groups.
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 UUID groups, got %d", len(parts))
	}

	// Check group lengths: 8-4-4-4-12 (standard UUID format)
	expectedLens := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLens[i] {
			t.Fatalf("group %d: expected length %d, got %d (%q)", i, expectedLens[i], len(part), part)
		}
	}

	// Third group should start with "7" (version).
	if parts[2][0] != '7' {
		t.Fatalf("third group should start with '7', got %q", string(parts[2][0]))
	}

	// Fourth group should start with "8", "9", "a", or "b" (variant).
	fourthFirst := parts[3][0]
	if fourthFirst != '8' && fourthFirst != '9' && fourthFirst != 'a' && fourthFirst != 'b' {
		t.Fatalf("fourth group should start with 8/9/a/b, got %q", string(fourthFirst))
	}
}

func TestRequestIDPropagated(t *testing.T) {
	existingID := "existing-id-12345"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	mw := RequestID()
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", existingID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id != existingID {
		t.Fatalf("expected propagated ID %q, got %q", existingID, id)
	}
}

func TestRequestIDInContext(t *testing.T) {
	var ctxID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = GetRequestID(r.Context())
	})

	mw := RequestID()
	h := mw(handler)

	expectedID := "my-test-request-id"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", expectedID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if ctxID != expectedID {
		t.Fatalf("expected context ID %q, got %q", expectedID, ctxID)
	}
}

func TestRequestIDCustomHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	customHeader := "X-Correlation-ID"
	mw := RequestID(RequestIDOptions{HeaderName: customHeader})
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	id := rec.Header().Get(customHeader)
	if id == "" {
		t.Fatalf("expected custom header %q to be set, got empty", customHeader)
	}

	// The default header should not be set.
	defaultID := rec.Header().Get("X-Request-ID")
	if defaultID != "" {
		t.Fatalf("expected default header to be empty, got %q", defaultID)
	}

	// Propagation with custom header.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(customHeader, "propagated-123")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	id2 := rec2.Header().Get(customHeader)
	if id2 != "propagated-123" {
		t.Fatalf("expected propagated custom header %q, got %q", "propagated-123", id2)
	}
}

func TestRequestIDCustomGenerator(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	counter := 0
	customGen := func() string {
		counter++
		return fmt.Sprintf("custom-%d", counter)
	}

	mw := RequestID(RequestIDOptions{Generator: customGen})
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id != "custom-1" {
		t.Fatalf("expected %q, got %q", "custom-1", id)
	}

	// Second request should produce "custom-2".
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	id2 := rec2.Header().Get("X-Request-ID")
	if id2 != "custom-2" {
		t.Fatalf("expected %q, got %q", "custom-2", id2)
	}
}

func TestRequestIDUniqueness(t *testing.T) {
	const n = 1000
	ids := make([]string, 0, n)
	seen := make(map[string]struct{}, n)

	for i := 0; i < n; i++ {
		id := generateUUIDv7()
		ids = append(ids, id)
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate UUID generated at index %d: %q", i, id)
		}
		seen[id] = struct{}{}

		// Verify format for every generated ID.
		if !uuidv7Pattern.MatchString(id) {
			t.Fatalf("ID at index %d does not match UUIDv7 format: %q", i, id)
		}
	}

	if len(seen) != n {
		t.Fatalf("expected %d unique IDs, got %d", n, len(seen))
	}
}

func BenchmarkRequestIDGenerate(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = generateUUIDv7()
	}
}

func BenchmarkRequestIDPassThrough(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	mw := RequestID()
	h := mw(handler)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", "benchmark-pass-through-id")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}
