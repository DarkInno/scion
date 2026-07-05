package ratelimit

import (
	"sync"
	"time"
)

// slidingWindowBucket holds the state for a sliding window rate limiter.
type slidingWindowBucket struct {
	timestamps []int64 // sorted ascending (oldest first)
}

// SlidingWindowLimiter implements rate limiting using the sliding window
// (sliding log) algorithm.
//
// Unlike the fixed window algorithm, the sliding window tracks the exact
// timestamp of each request. At any point in time, only requests within the
// last window duration are counted. This provides a smoother rate limit
// without boundary bursts.
//
// The sliding log is bounded by rate: at most rate timestamps are stored
// per key, since additional requests are denied.
type SlidingWindowLimiter struct {
	store  Store
	rate   int           // maximum requests per window
	window time.Duration // window duration
	mu     sync.Mutex    // protects the Get-Set sequence for atomicity
}

// NewSlidingWindowLimiter creates a new SlidingWindowLimiter.
//
// Parameters:
//   - store: the state store (must not be nil)
//   - rate: maximum requests per window (must be > 0)
//   - window: sliding window duration (must be > 0)
func NewSlidingWindowLimiter(store Store, rate int, window time.Duration) (*SlidingWindowLimiter, error) {
	if store == nil {
		return nil, ErrNilStore
	}
	if rate <= 0 {
		return nil, ErrInvalidRate
	}
	if window <= 0 {
		return nil, ErrInvalidWindow
	}
	return &SlidingWindowLimiter{
		store:  store,
		rate:   rate,
		window: window,
	}, nil
}

// Allow checks if a request with the given key is allowed under the rate limit.
// It returns a Result containing the outcome and metadata for rate limit headers.
func (l *SlidingWindowLimiter) Allow(key string) Result {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UnixNano()
	windowNanos := int64(l.window)
	cutoff := now - windowNanos

	var bucket *slidingWindowBucket
	if v, ok := l.store.Get(key); ok {
		if b, ok := v.(*slidingWindowBucket); ok {
			bucket = b
		}
	}
	if bucket == nil {
		bucket = &slidingWindowBucket{
			timestamps: make([]int64, 0, l.rate+1),
		}
	}

	// Remove expired timestamps (those outside the sliding window).
	// Use compaction to free the underlying array and prevent memory growth.
	i := 0
	for i < len(bucket.timestamps) && bucket.timestamps[i] <= cutoff {
		i++
	}
	if i > 0 {
		bucket.timestamps = append(bucket.timestamps[:0], bucket.timestamps[i:]...)
	}

	// Determine when the rate limit will reset:
	// when the oldest request in the window expires.
	var resetAtNanos int64
	if len(bucket.timestamps) > 0 {
		resetAtNanos = bucket.timestamps[0] + windowNanos
	} else {
		resetAtNanos = now + windowNanos
	}

	if len(bucket.timestamps) >= l.rate {
		l.store.Set(key, bucket)
		return Result{
			Allowed:    false,
			Limit:      l.rate,
			Remaining:  0,
			ResetAt:    resetAtNanos / int64(time.Second),
			RetryAfter: ceilDivSeconds(resetAtNanos - now),
		}
	}

	bucket.timestamps = append(bucket.timestamps, now)
	l.store.Set(key, bucket)

	remaining := l.rate - len(bucket.timestamps)
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
