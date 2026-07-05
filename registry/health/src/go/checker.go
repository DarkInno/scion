// Package health provides a dependency-free health checking framework for
// backend services. It supports HTTP ping, TCP dial and custom function
// checks, concurrent execution, result caching and standard liveness /
// readiness HTTP probes.
//
// The module uses only the Go standard library.
package health

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Result statuses for individual checks.
const (
	StatusPass = "pass"
	StatusFail = "fail"
)

// Aggregated statuses returned by RunChecks and the HTTP probes.
const (
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
	StatusReady     = "ready"
	StatusNotReady  = "not_ready"
)

// Defaults and policy limits.
const (
	DefaultTimeout  = 5 * time.Second
	DefaultCacheTTL = 5 * time.Second
	MaxNameLen      = 64
	MaxURLLen       = 2048

	// runSlack is added to the per-check timeout to form the overall run
	// budget. It absorbs scheduling/context-cancellation latency for
	// well-behaved checks while still bounding the probe when a check
	// misbehaves and ignores its context.
	runSlack = 200 * time.Millisecond
)

// Result is the outcome of a single check.
//
// Example JSON:
//
//	{"status":"pass","latency_ms":5}
//	{"status":"fail","error":"connection refused"}
type Result struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// PassResult builds a passing Result with the measured latency.
func PassResult(latency time.Duration) Result {
	return Result{Status: StatusPass, LatencyMs: latency.Milliseconds()}
}

// FailResult builds a failing Result from an error. A nil error is reported as
// "unknown error".
func FailResult(err error) Result {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}
	return Result{Status: StatusFail, Error: err.Error()}
}

// Check is a single named health probe.
type Check interface {
	Name() string
	Execute(ctx context.Context) Result
}

// cachedSnapshot holds a point-in-time aggregated result set.
type cachedSnapshot struct {
	status  string
	results map[string]Result
	at      time.Time
}

// HealthChecker manages a set of checks, runs them concurrently and caches the
// aggregated result to avoid probing backends too frequently.
type HealthChecker struct {
	mu       sync.RWMutex
	checks   []Check
	cache    *cachedSnapshot
	cacheTTL time.Duration
	timeout  time.Duration
}

// Option configures a HealthChecker.
type Option func(*HealthChecker)

// WithTimeout sets the per-check timeout (default 5s).
func WithTimeout(t time.Duration) Option {
	return func(hc *HealthChecker) {
		if t > 0 {
			hc.timeout = t
		}
	}
}

// WithCacheTTL sets the result cache TTL (default 5s). Pass 0 to disable
// caching entirely so every RunChecks call probes the backends.
func WithCacheTTL(ttl time.Duration) Option {
	return func(hc *HealthChecker) {
		hc.cacheTTL = ttl
	}
}

// New creates a HealthChecker applying the given options.
func New(opts ...Option) *HealthChecker {
	hc := &HealthChecker{
		timeout:  DefaultTimeout,
		cacheTTL: DefaultCacheTTL,
	}
	for _, o := range opts {
		o(hc)
	}
	return hc
}

// AddCheck registers a check. The check name is validated for length and CRLF
// and must be unique within the checker. It is safe for concurrent use.
func (hc *HealthChecker) AddCheck(c Check) error {
	if isNilCheck(c) {
		return fmt.Errorf("check is nil")
	}
	if err := ValidateName(c.Name()); err != nil {
		return err
	}
	hc.mu.Lock()
	defer hc.mu.Unlock()
	for _, existing := range hc.checks {
		if existing.Name() == c.Name() {
			return fmt.Errorf("duplicate check name: %s", c.Name())
		}
	}
	hc.checks = append(hc.checks, c)
	return nil
}

// Checks returns a snapshot copy of the registered checks.
func (hc *HealthChecker) Checks() []Check {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	out := make([]Check, len(hc.checks))
	copy(out, hc.checks)
	return out
}

// Liveness reports whether the process itself is alive. It performs no checks
// and is always considered passing.
func (hc *HealthChecker) Liveness() Result {
	return Result{Status: StatusPass}
}

// RunChecks executes all registered checks concurrently and returns the
// aggregated status together with per-check results. Results are cached for
// the configured TTL to avoid probing backends too frequently.
func (hc *HealthChecker) RunChecks(ctx context.Context) (string, map[string]Result) {
	if hc.cacheTTL > 0 {
		hc.mu.RLock()
		if hc.cache != nil && time.Since(hc.cache.at) < hc.cacheTTL {
			snap := hc.cache
			hc.mu.RUnlock()
			return snap.status, snap.results
		}
		hc.mu.RUnlock()
	}

	status, results := hc.runConcurrent(ctx)

	if hc.cacheTTL > 0 {
		hc.mu.Lock()
		hc.cache = &cachedSnapshot{status: status, results: results, at: time.Now()}
		hc.mu.Unlock()
	}
	return status, results
}

// runConcurrent fans out all checks, each bounded by its own timeout derived
// from the checker timeout. A buffered channel guarantees that goroutines
// never block on send, so even a misbehaving check that outlives its context
// cannot leak through the channel. An overall budget protects the probe from
// hanging when a check ignores its context.
func (hc *HealthChecker) runConcurrent(ctx context.Context) (string, map[string]Result) {
	hc.mu.RLock()
	checks := make([]Check, len(hc.checks))
	copy(checks, hc.checks)
	timeout := hc.timeout
	hc.mu.RUnlock()

	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	n := len(checks)
	results := make(map[string]Result, n)
	if n == 0 {
		return StatusHealthy, results
	}

	type item struct {
		name string
		r    Result
	}
	ch := make(chan item, n)

	for _, c := range checks {
		go func(c Check) {
			cctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			ch <- item{name: c.Name(), r: c.Execute(cctx)}
		}(c)
	}

	budget := time.NewTimer(timeout + runSlack)
	defer budget.Stop()

	timedOut := false
	for i := 0; i < n; i++ {
		select {
		case it := <-ch:
			results[it.name] = it.r
		case <-budget.C:
			timedOut = true
		}
		if timedOut {
			// Remaining uncollected checks are reported as timed out.
			for _, c := range checks {
				if _, ok := results[c.Name()]; !ok {
					results[c.Name()] = Result{Status: StatusFail, Error: "check timed out"}
				}
			}
			break
		}
	}

	status := StatusHealthy
	for _, r := range results {
		if r.Status != StatusPass {
			status = StatusUnhealthy
			break
		}
	}
	return status, results
}

// --- validation helpers ---

// isNilCheck reports whether c is an untyped-nil or typed-nil Check. A typed
// nil (e.g. (*CustomCheck)(nil) returned by a failed constructor) would
// otherwise panic on the Name() call, so it is detected defensively here.
func isNilCheck(c Check) bool {
	if c == nil {
		return true
	}
	v := reflect.ValueOf(c)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}

// ValidateName enforces the check-name policy: non-empty, at most MaxNameLen
// bytes and free of CR/LF characters (header injection prevention).
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("check name is empty")
	}
	if len(name) > MaxNameLen {
		return fmt.Errorf("check name too long: %d > %d", len(name), MaxNameLen)
	}
	if containsCRLF(name) {
		return fmt.Errorf("check name contains CRLF characters")
	}
	return nil
}

// containsCRLF reports whether s contains a CR or LF byte.
func containsCRLF(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}
