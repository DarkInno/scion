package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TraceContext returns a middleware that handles W3C Trace Context propagation.
// It parses incoming `traceparent` headers, injects response `traceparent`,
// and transparently passes `baggage` headers.
//
// Zero OpenTelemetry dependencies: only standard W3C trace context format.
// Compatible with any OTel-instrumented service downstream.
func TraceContext(opts ...TraceOptions) func(http.Handler) http.Handler {
	var opt TraceOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.HeaderName == "" {
		opt.HeaderName = "traceparent"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceparent := r.Header.Get(opt.HeaderName)

			if traceparent == "" {
				// Generate new trace context for the root request.
				traceparent = generateTraceParent()
			}

			// Validate format: version-traceid-spanid-flags
			// If invalid, generate a new one.
			traceID, spanID := parseTraceParent(traceparent)
			if traceID == "" || spanID == "" {
				traceparent = generateTraceParent()
				traceID, spanID = parseTraceParent(traceparent)
			}

			// Inject into response header for client correlation.
			w.Header().Set(opt.HeaderName, traceparent)

			// Store in context for downstream handlers.
			ctx := r.Context()
			ctx = context.WithValue(ctx, traceIDKey, traceID)
			ctx = context.WithValue(ctx, spanIDKey, spanID)
			ctx = context.WithValue(ctx, traceParentKey, traceparent)

			// Propagate baggage header if present.
			// Validate against CRLF to prevent header injection / DoS via panic.
			if baggage := r.Header.Get("baggage"); baggage != "" && isSafeHeaderValue(baggage) {
				ctx = context.WithValue(ctx, baggageKey, baggage)
				w.Header().Set("baggage", baggage)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTraceID extracts the trace ID from context.
// Returns empty string if not found.
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}

// GetSpanID extracts the span ID from context.
// Returns empty string if not found.
func GetSpanID(ctx context.Context) string {
	if id, ok := ctx.Value(spanIDKey).(string); ok {
		return id
	}
	return ""
}

// generateTraceParent creates a new W3C traceparent header value.
// Format: 00-<32hex traceid>-<16hex spanid>-<2hex flags>
func generateTraceParent() string {
	var traceID [16]byte
	var spanID [8]byte

	if _, err := io.ReadFull(rand.Reader, traceID[:]); err != nil {
		return fallbackTraceParent()
	}
	if _, err := io.ReadFull(rand.Reader, spanID[:]); err != nil {
		return fallbackTraceParent()
	}

	// Version: 00 (W3C trace context version 1)
	// Flags: 01 (sampled)
	return fmt.Sprintf("00-%032x-%016x-01", traceID, spanID)
}

// parseTraceParent extracts trace ID and span ID from a traceparent header.
// Returns empty strings if any part fails W3C validation.
// Expected format: 00-<32hex traceid>-<16hex spanid>-<2hex flags>
func parseTraceParent(tp string) (traceID, spanID string) {
	parts := strings.Split(tp, "-")
	if len(parts) != 4 {
		return "", ""
	}
	version := parts[0]
	traceID = parts[1]
	spanID = parts[2]
	flags := parts[3]

	// W3C trace context version 1 only supports version "00".
	if version != "00" {
		return "", ""
	}

	// Validate hex format and lengths.
	if len(traceID) != 32 || !isHex(traceID) {
		return "", ""
	}
	if len(spanID) != 16 || !isHex(spanID) {
		return "", ""
	}
	if len(flags) != 2 || !isHex(flags) {
		return "", ""
	}
	return traceID, spanID
}

// isHex checks if a string consists only of hex characters.
func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// fallbackTraceParent generates a deterministic traceparent when crypto/rand fails.
func fallbackTraceParent() string {
	ms := uint64(time.Now().UnixMilli())
	return fmt.Sprintf("00-%016x0000000000000000-%016x-01", ms, ms)
}

// isSafeHeaderValue checks that a string does not contain CRLF characters
// or other characters that would cause http.Header.Set to panic.
// This prevents header injection and DoS via crafted header values.
func isSafeHeaderValue(s string) bool {
	return !strings.ContainsAny(s, "\r\n\x00")
}
