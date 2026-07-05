package cache

import (
	"context"
	"sync"
	"time"
)

// Compile-time assertion that MemoryCache satisfies Store.
var _ Store[any] = (*MemoryCache[any])(nil)

// options configures a MemoryCache at construction time.
type options struct {
	maxEntries      int
	cleanupInterval time.Duration
}

// Option customises a MemoryCache.
type Option func(*options)

// WithMaxEntries overrides the maximum number of live entries. Values <= 0
// are ignored. The absolute minimum is 1.
func WithMaxEntries(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.maxEntries = n
		}
	}
}

// WithCleanupInterval overrides how often the background goroutine sweeps
// expired entries. Values <= 0 are ignored.
func WithCleanupInterval(d time.Duration) Option {
	return func(o *options) {
		if d > 0 {
			o.cleanupInterval = d
		}
	}
}

// MemoryCache is a concurrency-safe, in-memory implementation of Store. It
// keeps both generic values (typed V) and integer counters. Access recency
// is tracked with an LRU list so that, once maxEntries is reached, the
// least-recently-used value is evicted. Expired entries are removed lazily
// on access and periodically by a background goroutine that exits cleanly
// when Close is called.
type MemoryCache[V any] struct {
	mu              sync.RWMutex
	data            map[string]*lruNode[V]
	lru             *lruList[V]
	counters        map[string]int64
	maxEntries      int
	cleanupInterval time.Duration

	stopCh   chan struct{}
	done     chan struct{}
	stopOnce sync.Once
	closed   bool
}

// New creates and starts a MemoryCache. The returned cache owns a background
// cleanup goroutine; always call Close to stop it.
func New[V any](opts ...Option) *MemoryCache[V] {
	o := options{
		maxEntries:      defaultMaxEntries,
		cleanupInterval: time.Minute,
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxEntries < 1 {
		o.maxEntries = 1
	}
	c := &MemoryCache[V]{
		data:            make(map[string]*lruNode[V]),
		lru:             newLRUList[V](),
		counters:        make(map[string]int64),
		maxEntries:      o.maxEntries,
		cleanupInterval: o.cleanupInterval,
		stopCh:          make(chan struct{}),
		done:            make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

// cleanupLoop periodically removes expired entries. It terminates when
// stopCh is closed and then closes the done channel so Close can synchronise.
func (c *MemoryCache[V]) cleanupLoop() {
	defer close(c.done)
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.deleteExpired()
		}
	}
}

// deleteExpired removes every expired value entry. It is invoked under the
// write lock by the cleanup goroutine.
func (c *MemoryCache[V]) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	now := time.Now().UnixNano()
	for k, n := range c.data {
		if n.entry.Expiration != 0 && now > n.entry.Expiration {
			c.lru.remove(n)
			delete(c.data, k)
		}
	}
}

// evictIfNeeded removes least-recently-used entries until there is room for
// one additional entry. Values are evicted first (via the LRU list); if no
// values remain, an arbitrary counter is dropped. It must be called with the
// write lock held.
func (c *MemoryCache[V]) evictIfNeeded() {
	for len(c.data)+len(c.counters) >= c.maxEntries {
		if n := c.lru.back(); n != nil {
			c.lru.remove(n)
			delete(c.data, n.key)
			continue
		}
		// No values to evict: drop one counter instead.
		for k := range c.counters {
			delete(c.counters, k)
			break
		}
		// Guard against maxEntries == 0 / nothing to evict loops.
		if len(c.data) == 0 && len(c.counters) == 0 {
			return
		}
	}
}

// Set stores value under key with the given ttl. A ttl of 0 means the entry
// never expires. Updating an existing key refreshes both its value and its
// recency without growing the cache.
func (c *MemoryCache[V]) Set(ctx context.Context, key string, value V, ttl time.Duration) error {
	if err := validateKey(key); err != nil {
		return err
	}
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	if n, ok := c.data[key]; ok {
		n.entry = Entry[V]{Value: value, Expiration: exp}
		c.lru.moveFront(n)
		return nil
	}
	c.evictIfNeeded()
	n := &lruNode[V]{
		key:   key,
		entry: Entry[V]{Value: value, Expiration: exp},
	}
	c.data[key] = n
	c.lru.pushFront(n)
	return nil
}

// Get returns the value for key and marks it most-recently-used. Missing or
// expired keys return the zero value and false.
func (c *MemoryCache[V]) Get(ctx context.Context, key string) (V, bool) {
	var zero V
	if err := validateKey(key); err != nil {
		return zero, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return zero, false
	}
	n, ok := c.data[key]
	if !ok {
		return zero, false
	}
	if n.entry.Expired() {
		c.lru.remove(n)
		delete(c.data, key)
		return zero, false
	}
	c.lru.moveFront(n)
	return n.entry.Value, true
}

// Delete removes key (a value or a counter). It reports whether something
// was removed.
func (c *MemoryCache[V]) Delete(ctx context.Context, key string) bool {
	if err := validateKey(key); err != nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	n, ok := c.data[key]
	if ok {
		c.lru.remove(n)
		delete(c.data, key)
		return true
	}
	if _, ok := c.counters[key]; ok {
		delete(c.counters, key)
		return true
	}
	return false
}

// Has reports whether key exists and has not expired. It uses a read lock
// and does not touch recency.
func (c *MemoryCache[V]) Has(ctx context.Context, key string) bool {
	if err := validateKey(key); err != nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return false
	}
	if n, ok := c.data[key]; ok {
		return !n.entry.Expired()
	}
	_, ok := c.counters[key]
	return ok
}

// Incr atomically adds delta to the counter at key, creating it at 0 when
// absent, and returns the resulting value.
func (c *MemoryCache[V]) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	if err := validateKey(key); err != nil {
		return 0, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, ErrClosed
	}
	if _, exists := c.counters[key]; !exists {
		c.evictIfNeeded()
	}
	c.counters[key] += delta
	return c.counters[key], nil
}

// Decr atomically subtracts delta from the counter at key.
func (c *MemoryCache[V]) Decr(ctx context.Context, key string, delta int64) (int64, error) {
	return c.Incr(ctx, key, -delta)
}

// Len returns the total number of live values and counters.
func (c *MemoryCache[V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data) + len(c.counters)
}

// MaxEntries returns the configured entry cap.
func (c *MemoryCache[V]) MaxEntries() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxEntries
}

// Close stops the background cleanup goroutine and marks the cache closed.
// It is safe to call multiple times.
func (c *MemoryCache[V]) Close() error {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
	<-c.done
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return nil
}
