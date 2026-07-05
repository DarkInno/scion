package ratelimit

import (
	"errors"
	"testing"
	"time"
)

func TestFixedWindowLimiterAllowsThenDenies(t *testing.T) {
	limiter, err := NewFixedWindowLimiter(NewMemoryStore(), 1, time.Minute)
	if err != nil {
		t.Fatalf("NewFixedWindowLimiter: %v", err)
	}
	if result := limiter.Allow("client"); !result.Allowed || result.Remaining != 0 {
		t.Fatalf("first request = %+v", result)
	}
	if result := limiter.Allow("client"); result.Allowed || result.RetryAfter < 1 {
		t.Fatalf("second request should be denied: %+v", result)
	}
}

func TestFixedWindowLimiterRejectsInvalidConfig(t *testing.T) {
	if _, err := NewFixedWindowLimiter(nil, 1, time.Second); !errors.Is(err, ErrNilStore) {
		t.Fatalf("nil store = %v", err)
	}
	if _, err := NewFixedWindowLimiter(NewMemoryStore(), 0, time.Second); !errors.Is(err, ErrInvalidRate) {
		t.Fatalf("bad rate = %v", err)
	}
	if _, err := NewFixedWindowLimiter(NewMemoryStore(), 1, 0); !errors.Is(err, ErrInvalidWindow) {
		t.Fatalf("bad window = %v", err)
	}
}
