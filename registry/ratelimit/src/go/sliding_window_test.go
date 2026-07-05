package ratelimit

import (
	"errors"
	"testing"
	"time"
)

func TestSlidingWindowLimiterAllowsThenDenies(t *testing.T) {
	limiter, err := NewSlidingWindowLimiter(NewMemoryStore(), 1, time.Minute)
	if err != nil {
		t.Fatalf("NewSlidingWindowLimiter: %v", err)
	}
	if result := limiter.Allow("client"); !result.Allowed {
		t.Fatalf("first request = %+v", result)
	}
	if result := limiter.Allow("client"); result.Allowed || result.Limit != 1 {
		t.Fatalf("second request should be denied: %+v", result)
	}
}

func TestSlidingWindowLimiterRejectsInvalidConfig(t *testing.T) {
	if _, err := NewSlidingWindowLimiter(nil, 1, time.Second); !errors.Is(err, ErrNilStore) {
		t.Fatalf("nil store = %v", err)
	}
	if _, err := NewSlidingWindowLimiter(NewMemoryStore(), 0, time.Second); !errors.Is(err, ErrInvalidRate) {
		t.Fatalf("bad rate = %v", err)
	}
	if _, err := NewSlidingWindowLimiter(NewMemoryStore(), 1, 0); !errors.Is(err, ErrInvalidWindow) {
		t.Fatalf("bad window = %v", err)
	}
}
