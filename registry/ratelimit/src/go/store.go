package ratelimit

import (
	"errors"
	"sync"
	"time"
)

// Constants
const (
	// DefaultMaxBuckets is the default maximum number of buckets in a MemoryStore.
	// This prevents unbounded memory growth from unique keys.
	DefaultMaxBuckets = 10000

	// MaxKeyLength is the maximum allowed length for a rate limit key.
	// Keys exceeding this length are truncated by the middleware.
	MaxKeyLength = 256
)

// Errors
var (
	ErrInvalidRate     = errors.New("ratelimit: rate must be greater than 0")
	ErrInvalidWindow   = errors.New("ratelimit: window must be greater than 0")
	ErrInvalidCapacity = errors.New("ratelimit: capacity must be greater than 0")
	ErrNilStore        = errors.New("ratelimit: store must not be nil")
	ErrNilLimiter      = errors.New("ratelimit: limiter must not be nil")
)

// Result
// Result represents the outcome of a rate limit check.
type Result struct {
	// Allowed indicates whether the request is permitted.
	Allowed bool
	// Limit is the maximum number of requests allowed in the current window.
	Limit int
	// Remaining is the number of requests still allowed in the current window.
	Remaining int
	// ResetAt is the Unix timestamp (in seconds) when the limit will reset.
	ResetAt int64
	// RetryAfter is the number of seconds the client should wait before retrying.
	// This is 0 when Allowed is true.
	RetryAfter int
}

// Store Interface
// Store defines the interface for rate limit state storage.
// All methods must be safe for concurrent use.
//
// Implementations can range from in-memory maps (MemoryStore) to distributed
// caches (Redis, etc.). For distributed stores, ensure that Get-then-Set
// sequences are atomic (e.g., via Lua scripts in Redis).
type Store interface {
	// Get retrieves the value for the given key.
	// Returns the value and true if found, nil and false otherwise.
	Get(key string) (any, bool)
	// Set stores the value for the given key.
	Set(key string, value any)
	// Delete removes the value for the given key.
	Delete(key string)
}

// MemoryStore with LRU Eviction
// lruEntry is a node in the doubly-linked list used for LRU tracking.
type lruEntry struct {
	key   string
	value any
	prev  *lruEntry
	next  *lruEntry
}

// MemoryStore is an in-memory implementation of Store with LRU eviction.
//
// When the number of entries reaches maxBuckets, the least recently used
// entry is automatically evicted. This prevents unbounded memory growth
// from unique keys (e.g., spoofed client IPs or user IDs).
//
// MemoryStore is safe for concurrent use.
type MemoryStore struct {
	mu         sync.Mutex
	buckets    map[string]*lruEntry
	maxBuckets int
	head       *lruEntry // most recently used
	tail       *lruEntry // least recently used
}

// NewMemoryStore creates a new MemoryStore with the default max bucket limit.
func NewMemoryStore() *MemoryStore {
	return NewMemoryStoreWithLimit(DefaultMaxBuckets)
}

// NewMemoryStoreWithLimit creates a new MemoryStore with a custom max bucket limit.
// If maxBuckets <= 0, DefaultMaxBuckets is used.
func NewMemoryStoreWithLimit(maxBuckets int) *MemoryStore {
	if maxBuckets <= 0 {
		maxBuckets = DefaultMaxBuckets
	}
	return &MemoryStore{
		buckets:    make(map[string]*lruEntry),
		maxBuckets: maxBuckets,
	}
}

// Get retrieves the value for the given key and marks it as most recently used.
func (s *MemoryStore) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.buckets[key]
	if !ok {
		return nil, false
	}
	s.moveToFront(entry)
	return entry.value, true
}

// Set stores the value for the given key. If the store is at capacity,
// the least recently used entry is evicted.
func (s *MemoryStore) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.buckets[key]; ok {
		entry.value = value
		s.moveToFront(entry)
		return
	}

	entry := &lruEntry{key: key, value: value}
	s.buckets[key] = entry
	s.addToFront(entry)

	if len(s.buckets) > s.maxBuckets {
		s.evictTail()
	}
}

// Delete removes the value for the given key.
func (s *MemoryStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.buckets[key]; ok {
		s.remove(entry)
		delete(s.buckets, key)
	}
}

// Len returns the number of entries in the store.
func (s *MemoryStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.buckets)
}

// MaxBuckets returns the maximum number of buckets the store can hold.
func (s *MemoryStore) MaxBuckets() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxBuckets
}

// addToFront adds an entry to the front of the list (most recently used).
func (s *MemoryStore) addToFront(entry *lruEntry) {
	entry.prev = nil
	entry.next = s.head
	if s.head != nil {
		s.head.prev = entry
	}
	s.head = entry
	if s.tail == nil {
		s.tail = entry
	}
}

// remove unlinks an entry from the doubly-linked list.
func (s *MemoryStore) remove(entry *lruEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		s.head = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		s.tail = entry.prev
	}
	entry.prev = nil
	entry.next = nil
}

// moveToFront moves an existing entry to the front of the list.
func (s *MemoryStore) moveToFront(entry *lruEntry) {
	if s.head == entry {
		return
	}
	s.remove(entry)
	s.addToFront(entry)
}

// evictTail removes the least recently used entry (tail of the list).
func (s *MemoryStore) evictTail() {
	if s.tail == nil {
		return
	}
	key := s.tail.key
	s.remove(s.tail)
	delete(s.buckets, key)
}

// Helpers
// ceilDivSeconds converts a duration in nanoseconds to seconds using ceiling
// division, with a minimum return value of 1.
func ceilDivSeconds(nanos int64) int {
	sec := int64(time.Second)
	if nanos <= 0 {
		return 1
	}
	result := int(nanos / sec)
	if nanos%sec > 0 {
		result++
	}
	if result < 1 {
		result = 1
	}
	return result
}
