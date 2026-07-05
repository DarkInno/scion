package ratelimit

import (
	"errors"
	"testing"
)

func TestTokenBucketLimiterAllowsBurstThenDenies(t *testing.T) {
	limiter, err := NewTokenBucketLimiter(NewMemoryStore(), 1, 1)
	if err != nil {
		t.Fatalf("NewTokenBucketLimiter: %v", err)
	}
	if result := limiter.Allow("client"); !result.Allowed {
		t.Fatalf("first request = %+v", result)
	}
	if result := limiter.Allow("client"); result.Allowed || result.RetryAfter < 1 {
		t.Fatalf("second request should be denied: %+v", result)
	}
}

func TestTokenBucketLimiterRejectsInvalidConfig(t *testing.T) {
	if _, err := NewTokenBucketLimiter(nil, 1, 1); !errors.Is(err, ErrNilStore) {
		t.Fatalf("nil store = %v", err)
	}
	if _, err := NewTokenBucketLimiter(NewMemoryStore(), 0, 1); !errors.Is(err, ErrInvalidRate) {
		t.Fatalf("bad rate = %v", err)
	}
	if _, err := NewTokenBucketLimiter(NewMemoryStore(), 1, 0); !errors.Is(err, ErrInvalidCapacity) {
		t.Fatalf("bad capacity = %v", err)
	}
}
