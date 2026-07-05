package ratelimit

import (
	"math"
	"sync"
	"time"
)

// tokenBucket holds the state for a token bucket rate limiter.
type tokenBucket struct {
	tokens     float64 // current number of available tokens
	lastRefill int64   // unix nanoseconds of last refill
}

// TokenBucketLimiter implements rate limiting using the token bucket algorithm.
//
// Tokens are replenished continuously at rate tokens per second, up to a
// maximum of capacity tokens. Each request consumes one token. This allows
// bursts of up to capacity requests while maintaining an average rate of
// rate requests per second over time.
//
// A new bucket starts full (capacity tokens), allowing immediate bursts.
type TokenBucketLimiter struct {
	store    Store
	rate     float64 // tokens per second
	capacity float64 // maximum burst size
	mu       sync.Mutex
}

// NewTokenBucketLimiter creates a new TokenBucketLimiter.
//
// Parameters:
//   - store: the state store (must not be nil)
//   - rate: tokens per second (must be > 0)
//   - capacity: maximum burst size (must be > 0)
func NewTokenBucketLimiter(store Store, rate float64, capacity float64) (*TokenBucketLimiter, error) {
	if store == nil {
		return nil, ErrNilStore
	}
	if rate <= 0 {
		return nil, ErrInvalidRate
	}
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}
	return &TokenBucketLimiter{
		store:    store,
		rate:     rate,
		capacity: capacity,
	}, nil
}

// Allow checks if a request with the given key is allowed under the rate limit.
// It returns a Result containing the outcome and metadata for rate limit headers.
func (l *TokenBucketLimiter) Allow(key string) Result {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UnixNano()

	var bucket *tokenBucket
	if v, ok := l.store.Get(key); ok {
		if b, ok := v.(*tokenBucket); ok {
			bucket = b
		}
	}
	if bucket == nil {
		bucket = &tokenBucket{
			tokens:     l.capacity, // new buckets start full
			lastRefill: now,
		}
	}

	// Refill tokens based on elapsed time since last refill.
	// Guard against clock skew (negative elapsed time).
	elapsedNanos := now - bucket.lastRefill
	if elapsedNanos > 0 {
		elapsedSeconds := float64(elapsedNanos) / float64(time.Second)
		bucket.tokens = math.Min(l.capacity, bucket.tokens+elapsedSeconds*l.rate)
		bucket.lastRefill = now
	}

	limit := int(l.capacity)
	remaining := int(bucket.tokens)

	if bucket.tokens < 1 {
		// Not enough tokens; calculate when the next token will be available.
		needed := 1.0 - bucket.tokens
		waitSeconds := needed / l.rate
		waitNanos := int64(waitSeconds * float64(time.Second))
		resetAtNanos := now + waitNanos

		l.store.Set(key, bucket)
		return Result{
			Allowed:    false,
			Limit:      limit,
			Remaining:  remaining,
			ResetAt:    resetAtNanos / int64(time.Second),
			RetryAfter: ceilDivSeconds(waitNanos),
		}
	}

	// Consume one token
	bucket.tokens -= 1
	l.store.Set(key, bucket)

	remaining = int(bucket.tokens)

	// Reset time: when the bucket will be completely full again.
	var resetAtNanos int64
	tokensToFull := l.capacity - bucket.tokens
	if tokensToFull > 0 {
		waitNanos := int64((tokensToFull / l.rate) * float64(time.Second))
		resetAtNanos = now + waitNanos
	} else {
		resetAtNanos = now
	}

	return Result{
		Allowed:    true,
		Limit:      limit,
		Remaining:  remaining,
		ResetAt:    resetAtNanos / int64(time.Second),
		RetryAfter: 0,
	}
}
