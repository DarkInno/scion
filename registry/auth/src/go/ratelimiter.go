package auth

import (
	"sync"
	"time"
)

// MemoryRateLimiter is an in-memory rate limiter using a sliding window.
//
// It is suitable for single-instance deployments. For distributed deployments
// (multiple server instances behind a load balancer), replace this with a
// Redis-backed implementation.
//
// Usage:
//
//	rl := auth.NewMemoryRateLimiter(10, 15*time.Minute)
//	handler := auth.NewHandler(store, cfg).WithRateLimiter(rl)
type MemoryRateLimiter struct {
	maxRequests int
	window      time.Duration
	mu          sync.Mutex
	buckets     map[string]*bucket
}

// maxBuckets limits the number of tracked keys to prevent memory exhaustion.
// An attacker cannot exhaust memory by spamming requests with unique keys
// (different emails, different IPs) because old buckets are evicted.
const maxBuckets = 10000

// bucket tracks request timestamps for a single key.
type bucket struct {
	timestamps []time.Time
}

// NewMemoryRateLimiter creates a new in-memory rate limiter.
//
// maxRequests: maximum number of requests allowed within the window.
// window: time window for rate limiting (e.g. 15 * time.Minute).
func NewMemoryRateLimiter(maxRequests int, window time.Duration) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		maxRequests: maxRequests,
		window:      window,
		buckets:     make(map[string]*bucket),
	}
}

// Allow checks if the request identified by key is within the rate limit.
// Returns true if allowed, false if rate limit exceeded.
func (r *MemoryRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	b, ok := r.buckets[key]
	if !ok {
		// Evict expired buckets if we're at capacity to prevent memory exhaustion.
		if len(r.buckets) >= maxBuckets {
			r.cleanupLocked(cutoff)
		}
		// If cleanup didn't free enough space (all buckets still active),
		// evict the bucket with the oldest last-access time (LRU eviction).
		if len(r.buckets) >= maxBuckets {
			r.evictOldestLocked()
		}
		b = &bucket{timestamps: make([]time.Time, 0, r.maxRequests)}
		r.buckets[key] = b
	}

	// Remove timestamps outside the window.
	// Use a compaction approach: copy remaining timestamps to the front
	// of a new slice to free the underlying array and prevent memory growth.
	writeIdx := 0
	for _, t := range b.timestamps {
		if t.After(cutoff) {
			b.timestamps[writeIdx] = t
			writeIdx++
		}
	}
	b.timestamps = b.timestamps[:writeIdx]

	if len(b.timestamps) >= r.maxRequests {
		return false
	}

	b.timestamps = append(b.timestamps, now)
	return true
}

// Reset clears the rate limit state for a specific key.
// Useful for testing or manual unlock scenarios.
func (r *MemoryRateLimiter) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.buckets, key)
}

// Cleanup removes expired buckets to prevent memory growth.
// Call this periodically (e.g. via a background goroutine or on a schedule).
func (r *MemoryRateLimiter) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanupLocked(time.Now().Add(-r.window))
}

// cleanupLocked is the internal cleanup routine. Caller must hold the lock.
func (r *MemoryRateLimiter) cleanupLocked(cutoff time.Time) {
	for key, b := range r.buckets {
		if len(b.timestamps) == 0 || b.timestamps[len(b.timestamps)-1].Before(cutoff) {
			delete(r.buckets, key)
		}
	}
}

// evictOldestLocked removes the bucket with the oldest last-access timestamp.
// This is an LRU-style eviction used when cleanup can't free enough space
// (all buckets are still within their rate limit window).
// Caller must hold the lock.
func (r *MemoryRateLimiter) evictOldestLocked() {
	var oldestKey string
	var oldestTime time.Time
	first := true
	for key, b := range r.buckets {
		if len(b.timestamps) == 0 {
			delete(r.buckets, key)
			return
		}
		lastAccess := b.timestamps[len(b.timestamps)-1]
		if first || lastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = lastAccess
			first = false
		}
	}
	if !first {
		delete(r.buckets, oldestKey)
	}
}
