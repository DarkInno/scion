package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// RecoveryOptions configures the Recovery middleware.
type RecoveryOptions struct {
	// StackSize limits the depth of stack trace output (in frames).
	// 0 = default 32. Max 128.
	StackSize int

	// LogFunc is called when a panic is recovered. nil = slog.Error.
	// The first argument is the value passed to panic().
	LogFunc func(panicValue interface{}, stack []byte)

	// ResponseFunc is called after panic to send a response. nil = JSON 500.
	ResponseFunc func(w http.ResponseWriter)
}

// RecoveryDefaults returns the default RecoveryOptions.
func RecoveryDefaults() RecoveryOptions {
	return RecoveryOptions{StackSize: 32}
}

// TimeoutOptions configures the Timeout middleware.
type TimeoutOptions struct {
	// Timeout is the maximum request duration. 0 = default 30s. Max 5m.
	Timeout time.Duration

	// Message is the response body on timeout. Empty = default JSON.
	Message string
}

// TimeoutDefaults returns the default TimeoutOptions.
func TimeoutDefaults() TimeoutOptions {
	return TimeoutOptions{Timeout: 30 * time.Second}
}

// RequestIDOptions configures the RequestID middleware.
type RequestIDOptions struct {
	// HeaderName is the request ID header name. Empty = "X-Request-ID".
	HeaderName string

	// Generator is a custom ID generator. nil = UUIDv7.
	Generator func() string
}

// RequestIDDefaults returns the default RequestIDOptions.
func RequestIDDefaults() RequestIDOptions {
	return RequestIDOptions{HeaderName: "X-Request-ID"}
}

// AccessLogOptions configures the AccessLog middleware.
type AccessLogOptions struct {
	// SkipPaths is a list of paths that will not be logged (e.g., /health).
	SkipPaths []string

	// BufferSize is the async log channel buffer size. 0 = default 100. Max 10000.
	BufferSize int

	// DropOnFull: when true, drops log entries if channel is full.
	// When false, blocks the request (not recommended in production).
	DropOnFull bool
}

// AccessLogDefaults returns the default AccessLogOptions.
func AccessLogDefaults() AccessLogOptions {
	return AccessLogOptions{
		BufferSize: 100,
		DropOnFull: true,
	}
}

// CORSOptions configures the CORS middleware.
type CORSOptions struct {
	// AllowedOrigins is the list of allowed origins.
	// "*" means all (but NOT with AllowCredentials=true).
	AllowedOrigins []string

	// AllowedMethods is the list of allowed HTTP methods.
	AllowedMethods []string

	// AllowedHeaders is the list of allowed request headers.
	AllowedHeaders []string

	// ExposedHeaders is the list of headers exposed to the client.
	ExposedHeaders []string

	// AllowCredentials indicates whether credentials are allowed.
	// Security constraint: cannot be true with AllowedOrigins=["*"].
	AllowCredentials bool

	// MaxAge is the preflight cache duration in seconds. 0 = default 86400. Max 86400.
	MaxAge int
}

// CORSDefaults returns the default CORSOptions.
func CORSDefaults() CORSOptions {
	return CORSOptions{
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:         86400,
	}
}

// ProxyOptions configures the TrustedProxy middleware.
type ProxyOptions struct {
	// TrustedProxies is a list of trusted proxy CIDRs.
	TrustedProxies []string

	// ProxyCount is the number of trusted proxy layers. 0 = trust none.
	// If both set, TrustedProxies takes precedence.
	ProxyCount int
}

// ProxyDefaults returns the default ProxyOptions.
func ProxyDefaults() ProxyOptions {
	return ProxyOptions{}
}

// BodyLimitOptions configures the BodyLimit middleware.
type BodyLimitOptions struct {
	// MaxSize is the maximum request body size in bytes. 0 = default 1MB. Max 100MB.
	MaxSize int64
}

// BodyLimitDefaults returns the default BodyLimitOptions.
func BodyLimitDefaults() BodyLimitOptions {
	return BodyLimitOptions{MaxSize: 1 << 20}
}

// TraceOptions configures the TraceContext middleware.
type TraceOptions struct {
	// HeaderName is the traceparent header name. Empty = "traceparent".
	HeaderName string
}

// TraceDefaults returns the default TraceOptions.
func TraceDefaults() TraceOptions {
	return TraceOptions{HeaderName: "traceparent"}
}

// --- Env parsing helpers ---

// ParseEnvDuration reads a time.Duration from an env var (supports "30s", "5m", etc).
func ParseEnvDuration(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultVal
	}
	return d
}

// ParseEnvString reads a string from an env var.
func ParseEnvString(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// ParseEnvInt reads an int from an env var with min/max bounds.
func ParseEnvInt(key string, defaultVal, min, max int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

// ParseEnvStringSlice reads a comma-separated string list from an env var.
func ParseEnvStringSlice(key string, defaultVal []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
