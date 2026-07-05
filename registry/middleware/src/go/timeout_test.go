package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestTimeoutNormal(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mw := Timeout(TimeoutOptions{Timeout: 1 * time.Second})
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", rec.Body.String())
	}
}

func TestTimeoutExceeded(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should not see this"))
	})

	mw := Timeout(TimeoutOptions{Timeout: 50 * time.Millisecond})
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status 504, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}
	if body["error"] != "request timeout" {
		t.Fatalf("expected error %q, got %q", "request timeout", body["error"])
	}
}

func TestTimeoutContextPropagation(t *testing.T) {
	var detected bool
	var mu sync.Mutex

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		select {
		case <-r.Context().Done():
			detected = true
		case <-time.After(3 * time.Second):
		}
	})

	mw := Timeout(TimeoutOptions{Timeout: 50 * time.Millisecond})
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	mu.Lock()
	defer mu.Unlock()
	if !detected {
		t.Fatal("handler did not detect context cancellation on timeout")
	}
}

func TestTimeoutCustomMessage(t *testing.T) {
	customMsg := "custom gateway timeout error"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	})

	mw := Timeout(TimeoutOptions{
		Timeout: 50 * time.Millisecond,
		Message: customMsg,
	})
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status 504, got %d", rec.Code)
	}
	if rec.Body.String() != customMsg {
		t.Fatalf("expected body %q, got %q", customMsg, rec.Body.String())
	}
}

func TestTimeoutMaxClamp(t *testing.T) {
	// Use a channel to capture the actual deadline from inside the handler.
	deadlineCh := make(chan time.Duration, 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if ok {
			remaining := time.Until(deadline)
			deadlineCh <- remaining
		} else {
			deadlineCh <- 0
		}
	})

	// Set timeout to 10 minutes, should be clamped to 5 minutes.
	mw := Timeout(TimeoutOptions{Timeout: 10 * time.Minute})
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	remaining := <-deadlineCh
	// The remaining time should be close to 5 minutes (account for overhead).
	maxExpected := 5*time.Minute + 100*time.Millisecond
	if remaining > maxExpected {
		t.Fatalf("timeout was not clamped to 5min; remaining %v > max expected %v", remaining, maxExpected)
	}
	if remaining < 4*time.Minute {
		t.Fatalf("remaining time %v is unexpectedly short, expected ~5min", remaining)
	}
}

func TestTimeoutDefaultTimeout(t *testing.T) {
	deadlineCh := make(chan time.Duration, 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if ok {
			remaining := time.Until(deadline)
			deadlineCh <- remaining
		} else {
			deadlineCh <- 0
		}
	})

	// Timeout=0 should default to 30s.
	mw := Timeout(TimeoutOptions{Timeout: 0})
	h := mw(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	remaining := <-deadlineCh
	if remaining <= 0 {
		t.Fatal("expected a positive deadline remaining, got 0 (no deadline set)")
	}
	// Should be close to 30 seconds.
	if remaining < 29*time.Second || remaining > 31*time.Second {
		t.Fatalf("expected default timeout of ~30s, got remaining %v", remaining)
	}
}

func TestTimeoutRace(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a handler that writes and checks context concurrently.
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case <-r.Context().Done():
				case <-time.After(2 * time.Second):
				}
			}()
		}
		wg.Wait()
	})

	mw := Timeout(TimeoutOptions{Timeout: 50 * time.Millisecond})
	h := mw(handler)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
		}()
	}
	wg.Wait()
	// If this test passes with -race, no data race was detected.
}
