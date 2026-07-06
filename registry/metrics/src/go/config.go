// Package metrics provides Prometheus HTTP metrics for copy-paste Go services.
package metrics

import (
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// Options controls metric names, labels, and cardinality limits.
type Options struct {
	Namespace    string
	Subsystem    string
	MaxRouteLen  int
	MaxMethodLen int
	MaxRoutes    int
	Buckets      []float64
}

// Defaults returns safe HTTP metric defaults.
func Defaults() Options {
	return Options{
		Namespace:    "scion",
		Subsystem:    "http",
		MaxRouteLen:  128,
		MaxMethodLen: 16,
		MaxRoutes:    128,
		Buckets:      prometheus.DefBuckets,
	}
}

// FromEnv reads options from environment variables.
//
// Supported variables:
//   - METRICS_NAMESPACE
//   - METRICS_SUBSYSTEM
//   - METRICS_MAX_ROUTE_LEN
//   - METRICS_MAX_METHOD_LEN
//   - METRICS_MAX_ROUTES
func FromEnv() Options {
	o := Defaults()
	if v := os.Getenv("METRICS_NAMESPACE"); v != "" {
		o.Namespace = v
	}
	if v := os.Getenv("METRICS_SUBSYSTEM"); v != "" {
		o.Subsystem = v
	}
	if v := os.Getenv("METRICS_MAX_ROUTE_LEN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			o.MaxRouteLen = n
		}
	}
	if v := os.Getenv("METRICS_MAX_METHOD_LEN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			o.MaxMethodLen = n
		}
	}
	if v := os.Getenv("METRICS_MAX_ROUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			o.MaxRoutes = n
		}
	}
	return o
}

func (o Options) normalize() Options {
	d := Defaults()
	o.Namespace = safeMetricPart(o.Namespace, d.Namespace)
	o.Subsystem = safeMetricPart(o.Subsystem, d.Subsystem)
	if o.MaxRouteLen <= 0 {
		o.MaxRouteLen = d.MaxRouteLen
	}
	if o.MaxRouteLen > 512 {
		o.MaxRouteLen = 512
	}
	if o.MaxMethodLen <= 0 {
		o.MaxMethodLen = d.MaxMethodLen
	}
	if o.MaxMethodLen > 32 {
		o.MaxMethodLen = 32
	}
	if o.MaxRoutes <= 0 {
		o.MaxRoutes = d.MaxRoutes
	}
	if o.MaxRoutes > 10000 {
		o.MaxRoutes = 10000
	}
	if len(o.Buckets) == 0 {
		o.Buckets = d.Buckets
	}
	return o
}

func safeMetricPart(value, fallback string) string {
	if containsUnsafe(value) {
		return fallback
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return fallback
	}
	return value
}
