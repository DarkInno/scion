package metrics

import (
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const (
	unknownRoute  = "unknown"
	invalidRoute  = "invalid"
	overflowRoute = "overflow"
)

// Metrics owns an isolated Prometheus registry and HTTP metric collectors.
type Metrics struct {
	opts Options

	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge

	mu                 sync.Mutex
	routes             map[string]struct{}
	defaultsRegistered bool
}

// New creates an isolated metrics registry. It does not use the global
// Prometheus registry, so copied modules do not conflict with application code.
func New(opts ...Options) (*Metrics, error) {
	opt := Defaults()
	if len(opts) > 0 {
		opt = opts[0]
	}
	opt = opt.normalize()
	reg := prometheus.NewRegistry()
	m := &Metrics{
		opts:     opt,
		registry: reg,
		routes:   make(map[string]struct{}),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: opt.Namespace,
			Subsystem: opt.Subsystem,
			Name:      "requests_total",
			Help:      "Total HTTP requests by method, route, and status code.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: opt.Namespace,
			Subsystem: opt.Subsystem,
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds by method, route, and status code.",
			Buckets:   opt.Buckets,
		}, []string{"method", "route", "status"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: opt.Namespace,
			Subsystem: opt.Subsystem,
			Name:      "requests_in_flight",
			Help:      "Current in-flight HTTP requests.",
		}),
	}
	if err := reg.Register(m.requests); err != nil {
		return nil, err
	}
	if err := reg.Register(m.duration); err != nil {
		return nil, err
	}
	if err := reg.Register(m.inFlight); err != nil {
		return nil, err
	}
	return m, nil
}

// RegisterDefaults registers Go runtime and process collectors once.
func (m *Metrics) RegisterDefaults() error {
	if m == nil {
		return errors.New("metrics: Metrics is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.defaultsRegistered {
		return nil
	}
	if err := m.registry.Register(collectors.NewGoCollector()); err != nil {
		return err
	}
	if err := m.registry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return err
	}
	m.defaultsRegistered = true
	return nil
}

// Registry returns the isolated Prometheus registry.
func (m *Metrics) Registry() *prometheus.Registry {
	if m == nil {
		return prometheus.NewRegistry()
	}
	return m.registry
}

func (m *Metrics) observe(method, route string, status int, seconds float64) {
	method = safeMethod(method, m.opts)
	route = m.safeRoute(route)
	statusLabel := strconv.Itoa(status)
	m.requests.WithLabelValues(method, route, statusLabel).Inc()
	m.duration.WithLabelValues(method, route, statusLabel).Observe(seconds)
}

func (m *Metrics) safeRoute(route string) string {
	if containsUnsafe(route) {
		route = invalidRoute
	}
	route = strings.TrimSpace(route)
	switch {
	case route == "":
		route = unknownRoute
	case len(route) > m.opts.MaxRouteLen, strings.Contains(route, "?"), strings.Contains(route, "://"):
		route = invalidRoute
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.routes[route]; ok {
		return route
	}
	if len(m.routes) >= m.opts.MaxRoutes {
		return overflowRoute
	}
	m.routes[route] = struct{}{}
	return route
}

func safeMethod(method string, opts Options) string {
	if containsUnsafe(method) {
		return "UNKNOWN"
	}
	method = strings.TrimSpace(method)
	if method == "" || len(method) > opts.MaxMethodLen {
		return "UNKNOWN"
	}
	for i := 0; i < len(method); i++ {
		c := method[i]
		if c >= 'A' && c <= 'Z' {
			continue
		}
		return "UNKNOWN"
	}
	return method
}

func containsUnsafe(value string) bool {
	return strings.ContainsAny(value, "\r\n\x00")
}
