package ratelimit

import (
	"testing"
	"time"
)

func TestStoreMemoryEvictsLeastRecentlyUsed(t *testing.T) {
	store := NewMemoryStoreWithLimit(2)
	store.Set("a", 1)
	store.Set("b", 2)
	if _, ok := store.Get("a"); !ok {
		t.Fatal("expected a")
	}
	store.Set("c", 3)
	if _, ok := store.Get("b"); ok {
		t.Fatal("b should have been evicted")
	}
	if store.Len() != 2 || store.MaxBuckets() != 2 {
		t.Fatalf("unexpected store bounds len=%d max=%d", store.Len(), store.MaxBuckets())
	}
	store.Delete("a")
	if _, ok := store.Get("a"); ok {
		t.Fatal("a should be deleted")
	}
}

func TestStoreCeilDivSeconds(t *testing.T) {
	if got := ceilDivSeconds(1500 * int64(time.Millisecond)); got != 2 {
		t.Fatalf("ceilDivSeconds = %d", got)
	}
	if got := ceilDivSeconds(0); got != 1 {
		t.Fatalf("ceilDivSeconds minimum = %d", got)
	}
}
