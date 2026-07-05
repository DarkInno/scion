// Package cache provides a generic, concurrency-safe, TTL-aware key/value
// cache with an LRU eviction policy and a pluggable Store interface.
//
// The package has zero external dependencies and relies only on the Go
// standard library.
package cache

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Security limits
// ---------------------------------------------------------------------------

const (
	// maxKeyLength is the maximum number of bytes allowed in a cache key.
	// Keys longer than this are rejected to prevent unbounded memory growth
	// and to keep index operations cheap.
	maxKeyLength = 256

	// defaultMaxEntries is the default upper bound on the number of live
	// entries (both values and counters) kept in a MemoryCache.
	defaultMaxEntries = 1000
)

// Sentinel errors returned by the cache API.
var (
	// ErrEmptyKey is returned when an operation is attempted with an empty key.
	ErrEmptyKey = errors.New("cache: empty key")
	// ErrKeyTooLong is returned when a key exceeds maxKeyLength bytes.
	ErrKeyTooLong = errors.New("cache: key too long")
	// ErrInvalidKey is returned when a key contains a carriage return, line
	// feed or null byte (CRLF / null injection guard).
	ErrInvalidKey = errors.New("cache: key contains invalid characters")
	// ErrClosed is returned when an operation is attempted after Close.
	ErrClosed = errors.New("cache: store is closed")
)

// validateKey enforces the key security policy:
//   - non-empty
//   - at most maxKeyLength bytes
//   - no CR (\r), LF (\n) or NUL (\x00) bytes, which could be used for
//     header/log injection or to truncate keys when persisted to line
//     oriented stores (e.g. Redis RESP protocol).
func validateKey(key string) error {
	if len(key) == 0 {
		return ErrEmptyKey
	}
	if len(key) > maxKeyLength {
		return ErrKeyTooLong
	}
	if strings.ContainsAny(key, "\r\n\x00") {
		return ErrInvalidKey
	}
	return nil
}

// ---------------------------------------------------------------------------
// Entry
// ---------------------------------------------------------------------------

// Entry is a cached value together with its expiration metadata.
type Entry[V any] struct {
	Value      V
	Expiration int64 // absolute unix-nano deadline; 0 means "never expires"
}

// Expired reports whether the entry has surpassed its TTL. An entry with a
// zero Expiration never expires.
func (e Entry[V]) Expired() bool {
	if e.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > e.Expiration
}

// ---------------------------------------------------------------------------
// Store interface
// ---------------------------------------------------------------------------

// Store is the abstraction implemented by every cache backend. It is generic
// over the value type V so that callers get type-safe access without any
// runtime casts. A concrete in-memory implementation is provided by
// MemoryCache; the interface is intentionally minimal so that adapters for
// Redis, Memcached or any other backend can be written independently.
type Store[V any] interface {
	// Get retrieves the value for key. The boolean result is false when the
	// key is missing or expired.
	Get(ctx context.Context, key string) (V, bool)
	// Set stores value under key with the given time-to-live. A ttl of 0
	// means the entry never expires.
	Set(ctx context.Context, key string, value V, ttl time.Duration) error
	// Delete removes key. It reports whether the key was present.
	Delete(ctx context.Context, key string) bool
	// Has reports whether key exists and has not expired. It does not update
	// the recency information used by the LRU policy.
	Has(ctx context.Context, key string) bool
	// Incr atomically adds delta to the integer counter stored at key,
	// creating it (initialised to 0) when absent, and returns the new value.
	Incr(ctx context.Context, key string, delta int64) (int64, error)
	// Decr atomically subtracts delta from the counter at key.
	Decr(ctx context.Context, key string, delta int64) (int64, error)
	// Close releases all resources owned by the store, including background
	// goroutines. Subsequent operations return ErrClosed.
	Close() error
}

// ---------------------------------------------------------------------------
// LRU doubly-linked list
// ---------------------------------------------------------------------------

// lruNode is a single element of the LRU list. The same node pointer is
// stored in the cache map so that recency updates are O(1).
type lruNode[V any] struct {
	key   string
	entry Entry[V]
	prev  *lruNode[V]
	next  *lruNode[V]
}

// lruList is a doubly-linked list ordered from most-recently-used (head) to
// least-recently-used (tail). It is not safe for concurrent use on its own;
// callers must hold the owning MemoryCache's mutex.
type lruList[V any] struct {
	head *lruNode[V]
	tail *lruNode[V]
}

// newLRUList returns an empty LRU list.
func newLRUList[V any]() *lruList[V] {
	return &lruList[V]{}
}

// pushFront inserts node at the head (most recently used position).
func (l *lruList[V]) pushFront(n *lruNode[V]) {
	n.prev = nil
	n.next = l.head
	if l.head != nil {
		l.head.prev = n
	}
	l.head = n
	if l.tail == nil {
		l.tail = n
	}
}

// remove unlinks node from the list.
func (l *lruList[V]) remove(n *lruNode[V]) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		l.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		l.tail = n.prev
	}
	n.prev = nil
	n.next = nil
}

// moveFront moves an already-linked node to the head.
func (l *lruList[V]) moveFront(n *lruNode[V]) {
	if l.head == n {
		return
	}
	l.remove(n)
	l.pushFront(n)
}

// back returns the least-recently-used node (tail) or nil when empty.
func (l *lruList[V]) back() *lruNode[V] {
	return l.tail
}
