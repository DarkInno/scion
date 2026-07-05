package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newIntCache(t *testing.T, opts ...Option) *MemoryCache[int] {
	t.Helper()
	c := New[int](opts...)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestSetAndGet(t *testing.T) {
	c := newIntCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "a", 1, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v, ok := c.Get(ctx, "a")
	if !ok || v != 1 {
		t.Fatalf("Get(a) = (%d, %v), want (1, true)", v, ok)
	}

	if _, ok := c.Get(ctx, "missing"); ok {
		t.Fatal("missing key should not be found")
	}
}

func TestGetMissingReturnsZeroValue(t *testing.T) {
	c := newIntCache(t)
	v, ok := c.Get(context.Background(), "nope")
	if ok {
		t.Fatal("expected ok=false for missing key")
	}
	if v != 0 {
		t.Fatalf("expected zero value, got %d", v)
	}
}

func TestUpdateExistingKey(t *testing.T) {
	c := newIntCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "k", 1, 0)
	_ = c.Set(ctx, "k", 2, 0)
	v, ok := c.Get(ctx, "k")
	if !ok || v != 2 {
		t.Fatalf("Get(k) = (%d, %v), want (2, true)", v, ok)
	}
	if c.Len() != 1 {
		t.Fatalf("Len = %d, want 1 (update must not grow cache)", c.Len())
	}
}

func TestDelete(t *testing.T) {
	c := newIntCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "k", 1, 0)

	if !c.Delete(ctx, "k") {
		t.Fatal("Delete existing key should return true")
	}
	if c.Delete(ctx, "k") {
		t.Fatal("Delete missing key should return false")
	}
	if c.Has(ctx, "k") {
		t.Fatal("Has should be false after delete")
	}
}

func TestHas(t *testing.T) {
	c := newIntCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "present", 42, 0)
	if !c.Has(ctx, "present") {
		t.Fatal("Has(present) = false, want true")
	}
	if c.Has(ctx, "absent") {
		t.Fatal("Has(absent) = true, want false")
	}
}

func TestTTLExpiration(t *testing.T) {
	c := newIntCache(t, WithCleanupInterval(10*time.Millisecond))
	ctx := context.Background()

	_ = c.Set(ctx, "short", 1, 30*time.Millisecond)
	_ = c.Set(ctx, "forever", 2, 0)

	if v, ok := c.Get(ctx, "short"); !ok || v != 1 {
		t.Fatalf("Get(short) before expiry = (%d, %v)", v, ok)
	}
	time.Sleep(60 * time.Millisecond)
	if _, ok := c.Get(ctx, "short"); ok {
		t.Fatal("short should have expired")
	}
	if v, ok := c.Get(ctx, "forever"); !ok || v != 2 {
		t.Fatalf("forever must persist, got (%d, %v)", v, ok)
	}
}

func TestTTLBoundary(t *testing.T) {
	c := newIntCache(t)
	ctx := context.Background()

	// ttl == 0 => never expires
	_ = c.Set(ctx, "zero", 1, 0)
	if !c.Has(ctx, "zero") {
		t.Fatal("zero-ttl entry should be present")
	}

	// Very short ttl; ensure it is present immediately and gone after.
	_ = c.Set(ctx, "tiny", 9, 15*time.Millisecond)
	if _, ok := c.Get(ctx, "tiny"); !ok {
		t.Fatal("tiny should be present immediately after set")
	}
	time.Sleep(30 * time.Millisecond)
	if _, ok := c.Get(ctx, "tiny"); ok {
		t.Fatal("tiny should be expired")
	}
}

func TestBackgroundCleanupRemovesExpired(t *testing.T) {
	c := newIntCache(t, WithCleanupInterval(20*time.Millisecond))
	ctx := context.Background()
	_ = c.Set(ctx, "doomed", 1, 20*time.Millisecond)

	// Wait for at least one cleanup sweep past expiry.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if c.Len() == 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if c.Len() != 0 {
		t.Fatalf("background cleanup failed, Len=%d", c.Len())
	}
}

func TestLRUEviction(t *testing.T) {
	c := newIntCache(t, WithMaxEntries(3))
	ctx := context.Background()

	_ = c.Set(ctx, "a", 1, 0)
	_ = c.Set(ctx, "b", 2, 0)
	_ = c.Set(ctx, "c", 3, 0)

	// Touch "a" so it becomes most-recently-used.
	_, _ = c.Get(ctx, "a")

	// Insert "d" -> least recently used ("b") must be evicted.
	_ = c.Set(ctx, "d", 4, 0)

	if c.Has(ctx, "b") {
		t.Fatal("b should have been evicted as LRU")
	}
	for _, k := range []string{"a", "c", "d"} {
		if !c.Has(ctx, k) {
			t.Fatalf("key %s should still be present", k)
		}
	}
	if c.Len() != 3 {
		t.Fatalf("Len = %d, want 3", c.Len())
	}
}

func TestLRUUpdateDoesNotEvict(t *testing.T) {
	c := newIntCache(t, WithMaxEntries(2))
	ctx := context.Background()
	_ = c.Set(ctx, "a", 1, 0)
	_ = c.Set(ctx, "b", 2, 0)
	_ = c.Set(ctx, "a", 11, 0) // update existing
	if c.Len() != 2 {
		t.Fatalf("Len = %d, want 2", c.Len())
	}
	if v, _ := c.Get(ctx, "a"); v != 11 {
		t.Fatalf("a = %d, want 11", v)
	}
}

func TestIncrDecr(t *testing.T) {
	c := newIntCache(t)
	ctx := context.Background()

	v, err := c.Incr(ctx, "counter", 1)
	if err != nil {
		t.Fatalf("Incr: %v", err)
	}
	if v != 1 {
		t.Fatalf("Incr = %d, want 1", v)
	}
	v, _ = c.Incr(ctx, "counter", 5)
	if v != 6 {
		t.Fatalf("Incr = %d, want 6", v)
	}
	v, _ = c.Decr(ctx, "counter", 2)
	if v != 4 {
		t.Fatalf("Decr = %d, want 4", v)
	}
	v, _ = c.Incr(ctx, "counter", -10)
	if v != -6 {
		t.Fatalf("Incr negative = %d, want -6", v)
	}
}

func TestCounterDelete(t *testing.T) {
	c := newIntCache(t)
	ctx := context.Background()
	_, _ = c.Incr(ctx, "c", 3)
	if !c.Has(ctx, "c") {
		t.Fatal("counter should exist")
	}
	if !c.Delete(ctx, "c") {
		t.Fatal("Delete counter should return true")
	}
	if c.Has(ctx, "c") {
		t.Fatal("counter should be gone")
	}
}

func TestCounterEvictsValuesWhenFull(t *testing.T) {
	c := newIntCache(t, WithMaxEntries(2))
	ctx := context.Background()
	_ = c.Set(ctx, "v1", 1, 0)
	_ = c.Set(ctx, "v2", 2, 0)
	// Adding a counter while at cap must evict a value (LRU = v1).
	_, _ = c.Incr(ctx, "cnt", 1)
	if c.Has(ctx, "v1") {
		t.Fatal("v1 should have been evicted to make room for counter")
	}
	if !c.Has(ctx, "cnt") {
		t.Fatal("counter should be present")
	}
}

func TestStoreInterfaceCompliance(t *testing.T) {
	var s Store[string] = New[string](WithMaxEntries(5))
	defer s.Close()
	ctx := context.Background()
	if err := s.Set(ctx, "x", "y", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v, ok := s.Get(ctx, "x")
	if !ok || v != "y" {
		t.Fatalf("Get = (%q, %v)", v, ok)
	}
}

func TestOptionsValidation(t *testing.T) {
	// WithMaxEntries <= 0 should be ignored (default kept).
	c := newIntCache(t, WithMaxEntries(0))
	if c.MaxEntries() != defaultMaxEntries {
		t.Fatalf("MaxEntries = %d, want %d", c.MaxEntries(), defaultMaxEntries)
	}
	// WithMaxEntries == 1 enforced minimum via New.
	c2 := New[int](WithMaxEntries(1))
	defer c2.Close()
	if c2.MaxEntries() != 1 {
		t.Fatalf("MaxEntries = %d, want 1", c2.MaxEntries())
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := newIntCache(t, WithMaxEntries(500))
	ctx := context.Background()

	var wg sync.WaitGroup
	var errors atomic.Int64
	const goroutines = 50
	const ops = 200

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				key := fmt.Sprintf("k-%d-%d", id, j%20)
				switch j % 5 {
				case 0:
					if err := c.Set(ctx, key, j, 0); err != nil {
						errors.Add(1)
					}
				case 1:
					_, _ = c.Get(ctx, key)
				case 2:
					c.Delete(ctx, key)
				case 3:
					_, _ = c.Incr(ctx, "shared", 1)
				case 4:
					c.Has(ctx, key)
				}
			}
		}(i)
	}
	wg.Wait()

	if errors.Load() != 0 {
		t.Fatalf("got %d errors during concurrent access", errors.Load())
	}
	// Final state must be within the cap.
	if c.Len() > c.MaxEntries() {
		t.Fatalf("Len %d exceeds MaxEntries %d", c.Len(), c.MaxEntries())
	}
}

func TestCloseStopsGoroutine(t *testing.T) {
	c := New[int](WithCleanupInterval(5 * time.Millisecond))
	// Close must return and the done channel must be closed.
	_ = c.Close()
	select {
	case <-c.done:
	default:
		t.Fatal("done channel should be closed after Close")
	}
	// Double close must not panic.
	if err := c.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestOperationsAfterCloseReturnError(t *testing.T) {
	c := New[int]()
	ctx := context.Background()
	_ = c.Close()
	if err := c.Set(ctx, "k", 1, 0); err != ErrClosed {
		t.Fatalf("Set after close = %v, want ErrClosed", err)
	}
	if _, err := c.Incr(ctx, "k", 1); err != ErrClosed {
		t.Fatalf("Incr after close = %v, want ErrClosed", err)
	}
	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("Get after close should return false")
	}
}
