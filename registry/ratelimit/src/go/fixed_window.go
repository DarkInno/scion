package ratelimit

import (
	"sync"
	"time"
)

// fixedWindowBucket holds the state for a fixed window rate limiter.
type fixedWindowBucket struct {
	count       int
	windowStart int64 // unix nanoseconds
}

// FixedWindowLimiter implements rate limiting using the fixed window algorithm.
//
// Time is divided into fixed windows of the specified duration. Within each
// window, up to rate requests are allowed per key. When the window expires,
// the counter resets.
//
// This algorithm is simple and efficient but can allow bursts at window
// boundaries (e.g., rate requests at the end of one window and rate more at
// the start of the next).
type FixedWindowLimiter struct {
	store  Store
	rate   int           // maximum requests per window
	window time.Duration // window duration
	mu     sync.Mutex    // protects the Get-Set sequence for atomicity
}

// NewFixedWindowLimiter creates a new FixedWindowLimiter.
//
// Parameters:
//   - store: the state store (must not be nil)
//   - rate: maximum requests per window (must be > 0)
//   - window: duration of each window (must be > 0)
func NewFixedWindowLimiter(store Store, rate int, window time.Duration) (*FixedWindowLimiter, error) {
	if store == nil {
		return nil, ErrNilStore
	}
	if rate <= 0 {
		return nil, ErrInvalidRate
	}
	if window <= 0 {
		return nil, ErrInvalidWindow
	}
	return &FixedWindowLimiter{
		store:  store,
		rate:   rate,
		window: window,
	}, nil
}

// Allow checks if a request with the given key is allowed under the rate limit.
// It returns a Result containing the outcome and metadata for rate limit headers.
func (l *FixedWindowLimiter) Allow(key string) Result {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UnixNano()
	windowNanos := int64(l.window)

	var bucket *fixedWindowBucket
	if v, ok := l.store.Get(key); ok {
		if b, ok := v.(*fixedWindowBucket); ok {
			bucket = b
		}
	}
	if bucket == nil {
		bucket = &fixedWindowBucket{
			count:       0,
			windowStart: now,
		}
	}

	// Reset the window if it has expired
	if now-bucket.windowStart >= windowNanos {
		bucket.windowStart = now
		bucket.count = 0
	}

	resetAtNanos := bucket.windowStart + windowNanos

	if bucket.count >= l.rate {
		l.store.Set(key, bucket)
		return Result{
			Allowed:    false,
			Limit:      l.rate,
			Remaining:  0,
			ResetAt:    resetAtNanos / int64(time.Second),
			RetryAfter: ceilDivSeconds(resetAtNanos - now),
		}
	}

	bucket.count++
	l.store.Set(key, bucket)

	remaining := l.rate - bucket.count
	if remaining < 0 {
		remaining = 0
	}

	return Result{
		Allowed:    true,
		Limit:      l.rate,
		Remaining:  remaining,
		ResetAt:    resetAtNanos / int64(time.Second),
		RetryAfter: 0,
	}
}
