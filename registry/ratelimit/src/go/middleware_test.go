package ratelimit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Test Helpers
// ============================================================================

// okHandler returns a simple 200 OK handler for testing.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

// newRequestWithIP creates a test GET request with the given RemoteAddr.
func newRequestWithIP(ip string) *http.Request {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ip
	return req
}

// assertStatus checks that the response has the expected status code.
func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Errorf("status code = %d, want %d", rr.Code, want)
	}
}

// assertHeader checks that the response has the expected header value.
func assertHeader(t *testing.T, rr *httptest.ResponseRecorder, key, want string) {
	t.Helper()
	if got := rr.Header().Get(key); got != want {
		t.Errorf("header %s = %q, want %q", key, got, want)
	}
}

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNewFixedWindowLimiter(t *testing.T) {
	store := NewMemoryStore()

	// Valid
	l, err := NewFixedWindowLimiter(store, 10, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil limiter")
	}

	// Nil store
	_, err = NewFixedWindowLimiter(nil, 10, time.Second)
	if err != ErrNilStore {
		t.Errorf("nil store: expected ErrNilStore, got %v", err)
	}

	// Invalid rate
	_, err = NewFixedWindowLimiter(store, 0, time.Second)
	if err != ErrInvalidRate {
		t.Errorf("rate=0: expected ErrInvalidRate, got %v", err)
	}
	_, err = NewFixedWindowLimiter(store, -5, time.Second)
	if err != ErrInvalidRate {
		t.Errorf("rate=-5: expected ErrInvalidRate, got %v", err)
	}

	// Invalid window
	_, err = NewFixedWindowLimiter(store, 10, 0)
	if err != ErrInvalidWindow {
		t.Errorf("window=0: expected ErrInvalidWindow, got %v", err)
	}
	_, err = NewFixedWindowLimiter(store, 10, -time.Second)
	if err != ErrInvalidWindow {
		t.Errorf("window=-1s: expected ErrInvalidWindow, got %v", err)
	}
}

func TestNewSlidingWindowLimiter(t *testing.T) {
	store := NewMemoryStore()

	// Valid
	l, err := NewSlidingWindowLimiter(store, 10, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil limiter")
	}

	// Nil store
	_, err = NewSlidingWindowLimiter(nil, 10, time.Second)
	if err != ErrNilStore {
		t.Errorf("nil store: expected ErrNilStore, got %v", err)
	}

	// Invalid rate
	_, err = NewSlidingWindowLimiter(store, 0, time.Second)
	if err != ErrInvalidRate {
		t.Errorf("rate=0: expected ErrInvalidRate, got %v", err)
	}
	_, err = NewSlidingWindowLimiter(store, -1, time.Second)
	if err != ErrInvalidRate {
		t.Errorf("rate=-1: expected ErrInvalidRate, got %v", err)
	}

	// Invalid window
	_, err = NewSlidingWindowLimiter(store, 10, 0)
	if err != ErrInvalidWindow {
		t.Errorf("window=0: expected ErrInvalidWindow, got %v", err)
	}
}

func TestNewTokenBucketLimiter(t *testing.T) {
	store := NewMemoryStore()

	// Valid
	l, err := NewTokenBucketLimiter(store, 10, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil limiter")
	}

	// Nil store
	_, err = NewTokenBucketLimiter(nil, 10, 100)
	if err != ErrNilStore {
		t.Errorf("nil store: expected ErrNilStore, got %v", err)
	}

	// Invalid rate
	_, err = NewTokenBucketLimiter(store, 0, 100)
	if err != ErrInvalidRate {
		t.Errorf("rate=0: expected ErrInvalidRate, got %v", err)
	}
	_, err = NewTokenBucketLimiter(store, -1, 100)
	if err != ErrInvalidRate {
		t.Errorf("rate=-1: expected ErrInvalidRate, got %v", err)
	}

	// Invalid capacity
	_, err = NewTokenBucketLimiter(store, 10, 0)
	if err != ErrInvalidCapacity {
		t.Errorf("capacity=0: expected ErrInvalidCapacity, got %v", err)
	}
	_, err = NewTokenBucketLimiter(store, 10, -1)
	if err != ErrInvalidCapacity {
		t.Errorf("capacity=-1: expected ErrInvalidCapacity, got %v", err)
	}
}

// ============================================================================
// FixedWindowLimiter Tests
// ============================================================================

func TestFixedWindowLimiter_Allow(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewFixedWindowLimiter(store, 3, time.Second)

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		r := l.Allow("key1")
		if !r.Allowed {
			t.Errorf("request %d: expected allowed", i+1)
		}
		if r.Limit != 3 {
			t.Errorf("request %d: limit = %d, want 3", i+1, r.Limit)
		}
		if r.Remaining != 3-i-1 {
			t.Errorf("request %d: remaining = %d, want %d", i+1, r.Remaining, 3-i-1)
		}
		if r.RetryAfter != 0 {
			t.Errorf("request %d: retry_after = %d, want 0", i+1, r.RetryAfter)
		}
	}

	// 4th request should be denied
	r := l.Allow("key1")
	if r.Allowed {
		t.Error("4th request: expected denied")
	}
	if r.Remaining != 0 {
		t.Errorf("4th request: remaining = %d, want 0", r.Remaining)
	}
	if r.RetryAfter < 1 {
		t.Errorf("4th request: retry_after = %d, want >= 1", r.RetryAfter)
	}

	// Different key should be independent
	r = l.Allow("key2")
	if !r.Allowed {
		t.Error("different key: expected allowed")
	}
	if r.Remaining != 2 {
		t.Errorf("different key: remaining = %d, want 2", r.Remaining)
	}
}

func TestFixedWindowLimiter_WindowReset(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewFixedWindowLimiter(store, 2, 100*time.Millisecond)

	// Use up the limit
	l.Allow("key")
	l.Allow("key")

	// Should be denied
	r := l.Allow("key")
	if r.Allowed {
		t.Error("expected denied within window")
	}

	// Wait for window to reset
	time.Sleep(120 * time.Millisecond)

	// Should be allowed again
	r = l.Allow("key")
	if !r.Allowed {
		t.Error("expected allowed after window reset")
	}
	if r.Remaining != 1 {
		t.Errorf("remaining = %d, want 1", r.Remaining)
	}
}

func TestFixedWindowLimiter_ResetAt(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewFixedWindowLimiter(store, 5, time.Second)

	before := time.Now().Unix()
	r := l.Allow("test")
	after := time.Now().Unix()

	// ResetAt should be approximately now + 1 second
	if r.ResetAt < before+1 {
		t.Errorf("reset_at = %d, want >= %d", r.ResetAt, before+1)
	}
	if r.ResetAt > after+2 {
		t.Errorf("reset_at = %d, want <= %d", r.ResetAt, after+2)
	}
}

// ============================================================================
// SlidingWindowLimiter Tests
// ============================================================================

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewSlidingWindowLimiter(store, 3, time.Second)

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		r := l.Allow("key1")
		if !r.Allowed {
			t.Errorf("request %d: expected allowed", i+1)
		}
		if r.Limit != 3 {
			t.Errorf("request %d: limit = %d, want 3", i+1, r.Limit)
		}
		if r.Remaining != 3-i-1 {
			t.Errorf("request %d: remaining = %d, want %d", i+1, r.Remaining, 3-i-1)
		}
	}

	// 4th request should be denied
	r := l.Allow("key1")
	if r.Allowed {
		t.Error("4th request: expected denied")
	}
	if r.Remaining != 0 {
		t.Errorf("4th request: remaining = %d, want 0", r.Remaining)
	}
	if r.RetryAfter < 1 {
		t.Errorf("4th request: retry_after = %d, want >= 1", r.RetryAfter)
	}
}

func TestSlidingWindowLimiter_Sliding(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewSlidingWindowLimiter(store, 2, 200*time.Millisecond)

	// Request at t≈0: allowed (timestamps: [0])
	r := l.Allow("key")
	if !r.Allowed {
		t.Fatal("request at t=0: expected allowed")
	}

	// Wait 100ms, request at t≈100ms: allowed (timestamps: [0, 100])
	time.Sleep(100 * time.Millisecond)
	r = l.Allow("key")
	if !r.Allowed {
		t.Fatal("request at t=100ms: expected allowed")
	}

	// Third request at t≈100ms: denied (2 in window)
	r = l.Allow("key")
	if r.Allowed {
		t.Error("third request at t=100ms: expected denied")
	}

	// Wait until t≈210ms — the first request (at t≈0) has now expired
	// because cutoff = 210ms - 200ms = 10ms > 0.
	// The second request (at t≈100ms) is still in the window.
	// Count = 1, so a new request should be allowed.
	time.Sleep(110 * time.Millisecond)
	r = l.Allow("key")
	if !r.Allowed {
		t.Error("request after oldest expired: expected allowed (sliding window)")
	}
	// After this request, 2 timestamps are in the window (1 old + 1 new), so remaining = 0.
	// The key point is that the request was ALLOWED, which proves the sliding window
	// removed the expired timestamp (unlike a fixed window which would still deny).
	if r.Remaining != 0 {
		t.Errorf("remaining = %d, want 0 (2 in window after this request)", r.Remaining)
	}
}

func TestSlidingWindowLimiter_DifferentKeys(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewSlidingWindowLimiter(store, 1, time.Second)

	// key1: allowed then denied
	r1 := l.Allow("key1")
	if !r1.Allowed {
		t.Error("key1 first: expected allowed")
	}
	r2 := l.Allow("key1")
	if r2.Allowed {
		t.Error("key1 second: expected denied")
	}

	// key2: should be independent
	r3 := l.Allow("key2")
	if !r3.Allowed {
		t.Error("key2 first: expected allowed")
	}
}

// ============================================================================
// TokenBucketLimiter Tests
// ============================================================================

func TestTokenBucketLimiter_Allow(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewTokenBucketLimiter(store, 1.0, 3.0) // 1 token/sec, burst 3

	// Bucket starts full: 3 requests allowed
	for i := 0; i < 3; i++ {
		r := l.Allow("key1")
		if !r.Allowed {
			t.Errorf("request %d: expected allowed", i+1)
		}
		if r.Limit != 3 {
			t.Errorf("request %d: limit = %d, want 3", i+1, r.Limit)
		}
	}

	// 4th request should be denied (no tokens left)
	r := l.Allow("key1")
	if r.Allowed {
		t.Error("4th request: expected denied")
	}
	if r.Remaining != 0 {
		t.Errorf("4th request: remaining = %d, want 0", r.Remaining)
	}
	if r.RetryAfter < 1 {
		t.Errorf("4th request: retry_after = %d, want >= 1", r.RetryAfter)
	}
}

func TestTokenBucketLimiter_Burst(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewTokenBucketLimiter(store, 0.1, 5.0) // 0.1 token/sec, burst 5

	// Should allow 5 in quick succession (burst)
	allowed := 0
	for i := 0; i < 10; i++ {
		r := l.Allow("burst")
		if r.Allowed {
			allowed++
		}
	}
	if allowed != 5 {
		t.Errorf("burst allowed = %d, want 5", allowed)
	}
}

func TestTokenBucketLimiter_Refill(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewTokenBucketLimiter(store, 10.0, 2.0) // 10 tokens/sec, burst 2

	// Use up both tokens
	l.Allow("key")
	l.Allow("key")

	// Denied
	r := l.Allow("key")
	if r.Allowed {
		t.Error("expected denied after using all tokens")
	}

	// Wait for refill (0.1 sec at 10/sec = 1 token)
	time.Sleep(150 * time.Millisecond)

	// Should be allowed (1 token refilled)
	r = l.Allow("key")
	if !r.Allowed {
		t.Error("expected allowed after token refill")
	}
}

func TestTokenBucketLimiter_DifferentKeys(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewTokenBucketLimiter(store, 1.0, 1.0)

	// key1: allowed then denied
	r1 := l.Allow("key1")
	if !r1.Allowed {
		t.Error("key1 first: expected allowed")
	}
	r2 := l.Allow("key1")
	if r2.Allowed {
		t.Error("key1 second: expected denied")
	}

	// key2: independent bucket
	r3 := l.Allow("key2")
	if !r3.Allowed {
		t.Error("key2 first: expected allowed")
	}
}

// ============================================================================
// MemoryStore Tests
// ============================================================================

func TestMemoryStore_BasicOperations(t *testing.T) {
	s := NewMemoryStore()

	// Get on empty
	_, ok := s.Get("missing")
	if ok {
		t.Error("expected not found for missing key")
	}

	// Set and Get
	s.Set("a", 1)
	v, ok := s.Get("a")
	if !ok {
		t.Fatal("expected found for key 'a'")
	}
	if v != 1 {
		t.Errorf("value = %v, want 1", v)
	}

	// Update existing
	s.Set("a", 2)
	v, ok = s.Get("a")
	if !ok || v != 2 {
		t.Errorf("after update: value = %v, want 2", v)
	}

	// Delete
	s.Delete("a")
	_, ok = s.Get("a")
	if ok {
		t.Error("expected not found after delete")
	}
}

func TestMemoryStore_LRUEviction(t *testing.T) {
	s := NewMemoryStoreWithLimit(3)

	s.Set("a", 1)
	s.Set("b", 2)
	s.Set("c", 3)

	// Store is full (3 entries)
	if s.Len() != 3 {
		t.Fatalf("len = %d, want 3", s.Len())
	}

	// Access "a" to make it most recently used
	s.Get("a")

	// Add "d" — should evict "b" (least recently used)
	s.Set("d", 4)

	if s.Len() != 3 {
		t.Fatalf("len = %d, want 3", s.Len())
	}

	// "b" should be evicted
	_, ok := s.Get("b")
	if ok {
		t.Error("expected 'b' to be evicted")
	}

	// "a", "c", "d" should still exist
	for _, key := range []string{"a", "c", "d"} {
		_, ok := s.Get(key)
		if !ok {
			t.Errorf("expected key %q to exist", key)
		}
	}
}

func TestMemoryStore_LRUEvictionOrder(t *testing.T) {
	s := NewMemoryStoreWithLimit(5)

	// Add keys 0-4
	for i := 0; i < 5; i++ {
		s.Set("key"+strconv.Itoa(i), i)
	}

	// Access keys in order: 0, 2, 4 (making 1 and 3 the least recently used)
	s.Get("key0")
	s.Get("key2")
	s.Get("key4")

	// Add key5 — should evict key1 (oldest unused)
	s.Set("key5", 5)
	_, ok := s.Get("key1")
	if ok {
		t.Error("expected key1 to be evicted")
	}

	// Add key6 — should evict key3
	s.Set("key6", 6)
	_, ok = s.Get("key3")
	if ok {
		t.Error("expected key3 to be evicted")
	}

	// Keys 0, 2, 4, 5, 6 should exist
	for _, key := range []string{"key0", "key2", "key4", "key5", "key6"} {
		_, ok := s.Get(key)
		if !ok {
			t.Errorf("expected key %q to exist", key)
		}
	}
}

// ============================================================================
// Key Function Tests
// ============================================================================

func TestKeyByIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		want       string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For single",
			remoteAddr: "10.0.0.1:80",
			xff:        "203.0.113.5",
			want:       "203.0.113.5",
		},
		{
			name:       "X-Forwarded-For multiple",
			remoteAddr: "10.0.0.1:80",
			xff:        "203.0.113.5, 70.41.3.18, 150.172.238.178",
			want:       "203.0.113.5",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "10.0.0.1:80",
			xri:        "198.51.100.3",
			want:       "198.51.100.3",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr: "10.0.0.1:80",
			xff:        "203.0.113.5",
			xri:        "198.51.100.3",
			want:       "203.0.113.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequestWithIP(tt.remoteAddr)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			got := KeyByIP(req)
			if got != tt.want {
				t.Errorf("KeyByIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKeyByUserID(t *testing.T) {
	fn := KeyByUserID("X-User-ID")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-ID", "user123")
	if got := fn(req); got != "user123" {
		t.Errorf("got %q, want %q", got, "user123")
	}

	// Missing header
	req2 := httptest.NewRequest("GET", "/", nil)
	if got := fn(req2); got != "" {
		t.Errorf("missing header: got %q, want empty", got)
	}
}

func TestKeyGlobal(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if got := KeyGlobal(req); got != "global" {
		t.Errorf("got %q, want %q", got, "global")
	}
}

func TestKeyByCustom(t *testing.T) {
	fn := KeyByCustom(func(r *http.Request) string {
		return r.Method + ":" + r.URL.Path
	})

	req := httptest.NewRequest("POST", "/api/v1/users", nil)
	if got := fn(req); got != "POST:/api/v1/users" {
		t.Errorf("got %q, want %q", got, "POST:/api/v1/users")
	}
}

// ============================================================================
// Middleware Tests
// ============================================================================

func TestMiddleware_FixedWindow_AllowsThenDenies(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewFixedWindowLimiter(store, 2, time.Second)
	mw := Middleware(limiter, KeyByIP)
	handler := mw(okHandler())

	// First 2 requests: 200
	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		req := newRequestWithIP("1.2.3.4:1234")
		handler.ServeHTTP(rr, req)
		assertStatus(t, rr, http.StatusOK)
	}

	// 3rd request: 429
	rr := httptest.NewRecorder()
	req := newRequestWithIP("1.2.3.4:1234")
	handler.ServeHTTP(rr, req)
	assertStatus(t, rr, http.StatusTooManyRequests)
}

func TestMiddleware_SlidingWindow(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewSlidingWindowLimiter(store, 2, time.Second)
	mw := Middleware(limiter, KeyByIP)
	handler := mw(okHandler())

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		req := newRequestWithIP("5.6.7.8:9999")
		handler.ServeHTTP(rr, req)
		assertStatus(t, rr, http.StatusOK)
	}

	rr := httptest.NewRecorder()
	req := newRequestWithIP("5.6.7.8:9999")
	handler.ServeHTTP(rr, req)
	assertStatus(t, rr, http.StatusTooManyRequests)
}

func TestMiddleware_TokenBucket(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewTokenBucketLimiter(store, 1.0, 2.0)
	mw := Middleware(limiter, KeyByIP)
	handler := mw(okHandler())

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		req := newRequestWithIP("9.8.7.6:5555")
		handler.ServeHTTP(rr, req)
		assertStatus(t, rr, http.StatusOK)
	}

	rr := httptest.NewRecorder()
	req := newRequestWithIP("9.8.7.6:5555")
	handler.ServeHTTP(rr, req)
	assertStatus(t, rr, http.StatusTooManyRequests)
}

func TestMiddleware_HeadersOnAllowed(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewFixedWindowLimiter(store, 10, time.Second)
	mw := Middleware(limiter, KeyByIP)
	handler := mw(okHandler())

	rr := httptest.NewRecorder()
	req := newRequestWithIP("1.1.1.1:80")
	handler.ServeHTTP(rr, req)

	assertStatus(t, rr, http.StatusOK)
	assertHeader(t, rr, HeaderLimit, "10")
	assertHeader(t, rr, HeaderRemaining, "9")
	assertHeader(t, rr, HeaderReset, strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))

	// Retry-After should NOT be set on allowed responses
	if h := rr.Header().Get(HeaderRetryAfter); h != "" {
		t.Errorf("Retry-After should not be set on allowed response, got %q", h)
	}
}

func TestMiddleware_HeadersOnDenied(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewFixedWindowLimiter(store, 1, time.Second)
	mw := Middleware(limiter, KeyByIP)
	handler := mw(okHandler())

	// Use up the limit
	rr1 := httptest.NewRecorder()
	req1 := newRequestWithIP("2.2.2.2:80")
	handler.ServeHTTP(rr1, req1)
	assertStatus(t, rr1, http.StatusOK)

	// Denied request
	rr2 := httptest.NewRecorder()
	req2 := newRequestWithIP("2.2.2.2:80")
	handler.ServeHTTP(rr2, req2)

	assertStatus(t, rr2, http.StatusTooManyRequests)
	assertHeader(t, rr2, HeaderLimit, "1")
	assertHeader(t, rr2, HeaderRemaining, "0")

	retryAfter := rr2.Header().Get(HeaderRetryAfter)
	if retryAfter == "" {
		t.Error("expected Retry-After header on denied response")
	}
	ra, err := strconv.Atoi(retryAfter)
	if err != nil {
		t.Fatalf("invalid Retry-After value %q: %v", retryAfter, err)
	}
	if ra < 1 {
		t.Errorf("Retry-After = %d, want >= 1", ra)
	}

	// Verify Reset header is present
	if rr2.Header().Get(HeaderReset) == "" {
		t.Error("expected X-RateLimit-Reset header")
	}
}

func TestMiddleware_429ResponseBody(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewFixedWindowLimiter(store, 1, time.Second)
	mw := Middleware(limiter, KeyByIP)
	handler := mw(okHandler())

	// Use up the limit
	handler.ServeHTTP(httptest.NewRecorder(), newRequestWithIP("3.3.3.3:80"))

	// Denied request
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, newRequestWithIP("3.3.3.3:80"))

	assertStatus(t, rr, http.StatusTooManyRequests)
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body rateLimitResponse
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body.Error != "Too Many Requests" {
		t.Errorf("error = %q, want %q", body.Error, "Too Many Requests")
	}
	if body.RetryAfter < 1 {
		t.Errorf("retry_after = %d, want >= 1", body.RetryAfter)
	}
}

func TestMiddleware_DifferentKeysIndependent(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewFixedWindowLimiter(store, 1, time.Second)
	mw := Middleware(limiter, KeyByIP)
	handler := mw(okHandler())

	// IP 1: allowed
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, newRequestWithIP("10.0.0.1:1234"))
	assertStatus(t, rr1, http.StatusOK)

	// IP 1: denied (limit reached)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, newRequestWithIP("10.0.0.1:1234"))
	assertStatus(t, rr2, http.StatusTooManyRequests)

	// IP 2: allowed (different key)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, newRequestWithIP("10.0.0.2:5678"))
	assertStatus(t, rr3, http.StatusOK)
}

func TestMiddleware_DefaultKeyFunc(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewFixedWindowLimiter(store, 1, time.Second)

	// nil keyFunc should default to KeyByIP
	mw := Middleware(limiter, nil)
	handler := mw(okHandler())

	rr := httptest.NewRecorder()
	req := newRequestWithIP("4.4.4.4:80")
	handler.ServeHTTP(rr, req)
	assertStatus(t, rr, http.StatusOK)

	// Same IP should be denied
	rr2 := httptest.NewRecorder()
	req2 := newRequestWithIP("4.4.4.4:80")
	handler.ServeHTTP(rr2, req2)
	assertStatus(t, rr2, http.StatusTooManyRequests)
}

func TestMiddleware_KeyByUserID(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewFixedWindowLimiter(store, 1, time.Second)
	mw := Middleware(limiter, KeyByUserID("X-User-ID"))
	handler := mw(okHandler())

	// User A: allowed
	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.Header.Set("X-User-ID", "userA")
	req1.RemoteAddr = "1.2.3.4:80"
	handler.ServeHTTP(rr1, req1)
	assertStatus(t, rr1, http.StatusOK)

	// User A: denied
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-User-ID", "userA")
	req2.RemoteAddr = "5.6.7.8:80" // different IP, same user
	handler.ServeHTTP(rr2, req2)
	assertStatus(t, rr2, http.StatusTooManyRequests)

	// User B: allowed (different key)
	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.Header.Set("X-User-ID", "userB")
	req3.RemoteAddr = "1.2.3.4:80" // same IP as user A, but different user
	handler.ServeHTTP(rr3, req3)
	assertStatus(t, rr3, http.StatusOK)
}

func TestMiddleware_GlobalKey(t *testing.T) {
	store := NewMemoryStore()
	limiter, _ := NewFixedWindowLimiter(store, 2, time.Second)
	mw := Middleware(limiter, KeyGlobal)
	handler := mw(okHandler())

	// Two different IPs but same global key
	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		req := newRequestWithIP("10.0.0." + strconv.Itoa(i+1) + ":80")
		handler.ServeHTTP(rr, req)
		assertStatus(t, rr, http.StatusOK)
	}

	// Third request (different IP) should be denied (global limit)
	rr := httptest.NewRecorder()
	req := newRequestWithIP("10.0.0.99:80")
	handler.ServeHTTP(rr, req)
	assertStatus(t, rr, http.StatusTooManyRequests)
}

// ============================================================================
// Concurrent Test (basic, detailed in pentest_test.go)
// ============================================================================

func TestFixedWindowLimiter_Concurrent(t *testing.T) {
	store := NewMemoryStore()
	l, _ := NewFixedWindowLimiter(store, 100, time.Second)

	var allowed, denied int64
	var wg sync.WaitGroup

	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := l.Allow("concurrent-key")
			if r.Allowed {
				atomic.AddInt64(&allowed, 1)
			} else {
				atomic.AddInt64(&denied, 1)
			}
		}()
	}
	wg.Wait()

	if allowed != 100 {
		t.Errorf("allowed = %d, want 100", allowed)
	}
	if denied != 400 {
		t.Errorf("denied = %d, want 400", denied)
	}
}
