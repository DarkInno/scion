package auth

import (
	"sync"
	"testing"
	"time"
)

func TestMemoryRateLimiter_Allow(t *testing.T) {
	rl := NewMemoryRateLimiter(3, time.Minute)
	key := "test-key"

	// First 3 should be allowed
	if !rl.Allow(key) {
		t.Error("expected 1st request to be allowed")
	}
	if !rl.Allow(key) {
		t.Error("expected 2nd request to be allowed")
	}
	if !rl.Allow(key) {
		t.Error("expected 3rd request to be allowed")
	}

	// 4th should be denied
	if rl.Allow(key) {
		t.Error("expected 4th request to be denied")
	}
}

func TestMemoryRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewMemoryRateLimiter(2, time.Minute)

	// Each key has its own bucket
	if !rl.Allow("key-a") {
		t.Error("expected key-a 1st request to be allowed")
	}
	if !rl.Allow("key-a") {
		t.Error("expected key-a 2nd request to be allowed")
	}
	if rl.Allow("key-a") {
		t.Error("expected key-a 3rd request to be denied")
	}

	// key-b should still be allowed
	if !rl.Allow("key-b") {
		t.Error("expected key-b 1st request to be allowed")
	}
}

func TestMemoryRateLimiter_Reset(t *testing.T) {
	rl := NewMemoryRateLimiter(1, time.Minute)
	key := "test-key"

	if !rl.Allow(key) {
		t.Error("expected 1st request to be allowed")
	}
	if rl.Allow(key) {
		t.Error("expected 2nd request to be denied")
	}

	rl.Reset(key)

	if !rl.Allow(key) {
		t.Error("expected request after reset to be allowed")
	}
}

func TestMemoryRateLimiter_WindowSliding(t *testing.T) {
	rl := NewMemoryRateLimiter(2, 100*time.Millisecond)
	key := "test-key"

	if !rl.Allow(key) {
		t.Error("expected 1st request to be allowed")
	}
	if !rl.Allow(key) {
		t.Error("expected 2nd request to be allowed")
	}
	if rl.Allow(key) {
		t.Error("expected 3rd request to be denied")
	}

	// Wait for window to pass
	time.Sleep(150 * time.Millisecond)

	if !rl.Allow(key) {
		t.Error("expected request after window to be allowed")
	}
}

func TestMemoryRateLimiter_Cleanup(t *testing.T) {
	rl := NewMemoryRateLimiter(2, 100*time.Millisecond)

	rl.Allow("key-a")
	rl.Allow("key-b")

	if len(rl.buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(rl.buckets))
	}

	// Wait for window to pass
	time.Sleep(150 * time.Millisecond)

	rl.Cleanup()

	if len(rl.buckets) != 0 {
		t.Errorf("expected 0 buckets after cleanup, got %d", len(rl.buckets))
	}
}

func TestMemoryRateLimiter_Concurrent(t *testing.T) {
	rl := NewMemoryRateLimiter(100, time.Minute)
	key := "concurrent-key"
	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex

	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow(key) {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != 100 {
		t.Errorf("expected exactly 100 allowed, got %d", allowed)
	}
}
