package middleware

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"sync"
)

var (
	debugOnce = sync.OnceValue(func() bool {
		return os.Getenv("DEBUG") == "true"
	})
)

// resetDebugCache recreates the OnceValue cache. Used only in tests.
func resetDebugCache() {
	debugOnce = sync.OnceValue(func() bool {
		return os.Getenv("DEBUG") == "true"
	})
}

// DebugMode checks if DEBUG mode is enabled via the DEBUG environment variable.
// The result is cached after the first call for efficiency.
func DebugMode() bool {
	return debugOnce()
}

// DumpRequest returns a middleware that logs the raw request (headers only)
// when DEBUG mode is enabled. No effect in production.
func DumpRequest(next http.Handler) http.Handler {
	enabled := DebugMode() // Check once at middleware creation, not per-request.
	if !enabled {
		return next // Short-circuit: return the original handler unwrapped.
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, false)
		if err == nil {
			slog.Debug("request dump", "dump", string(dump))
		}
		next.ServeHTTP(w, r)
	})
}

// DebugHandler returns a handler that serves runtime diagnostics.
// Only available when DEBUG=true and accessed from localhost.
// SECURITY: Access control is based on r.RemoteAddr (TCP source IP),
// NOT r.Host (which is client-controlled and can be spoofed).
func DebugHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !DebugMode() {
			http.Error(w, "debug mode disabled", http.StatusNotFound)
			return
		}

		// Only allow localhost access based on RemoteAddr (TCP source).
		// r.Host is client-controlled and must NOT be used for access control.
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		ip := net.ParseIP(host)
		if ip == nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// Allow only loopback addresses (127.0.0.0/8, ::1/128).
		if !ip.IsLoopback() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		info := map[string]interface{}{
			"debug_mode": true,
			"goroutines": runtime.NumGoroutine(),
			"go_version": runtime.Version(),
			"num_cpu":    runtime.NumCPU(),
			"request": map[string]string{
				"method": r.Method,
				"path":   r.URL.Path,
				"host":   r.Host,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
	})
}
