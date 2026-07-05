package health

import (
	"encoding/json"
	"net/http"
)

// healthResponse is the JSON body returned by the HTTP probes.
//
// Example (healthy):
//
//	{"status":"healthy","checks":{"database":{"status":"pass","latency_ms":5}}}
//
// Example (liveness, no checks):
//
//	{"status":"healthy"}
type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]Result `json:"checks,omitempty"`
}

// HealthHandler exposes a HealthChecker over HTTP. Mount the handlers on the
// standard probe paths, for example:
//
//	mux.HandleFunc("/health", h.Health)       // overall health (+ checks)
//	mux.HandleFunc("/live",  h.Liveness)      // liveness (process alive)
//	mux.HandleFunc("/ready", h.Readiness)     // readiness (+ checks)
type HealthHandler struct {
	checker *HealthChecker
}

// NewHealthHandler creates a HealthHandler backed by the given checker.
func NewHealthHandler(c *HealthChecker) *HealthHandler {
	return &HealthHandler{checker: c}
}

// Liveness reports process liveness. It performs no checks and always returns
// HTTP 200 while the process is running.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: StatusHealthy})
}

// Readiness runs all checks and reports readiness. HTTP 200 when every check
// passes, 503 otherwise.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	status, checks := h.checker.RunChecks(r.Context())
	code := http.StatusOK
	respStatus := StatusReady
	if status != StatusHealthy {
		code = http.StatusServiceUnavailable
		respStatus = StatusNotReady
	}
	writeJSON(w, code, healthResponse{Status: respStatus, Checks: checks})
}

// Health runs all checks and reports the overall health. HTTP 200 when healthy,
// 503 otherwise.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	status, checks := h.checker.RunChecks(r.Context())
	code := http.StatusOK
	if status != StatusHealthy {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, healthResponse{Status: status, Checks: checks})
}

// writeJSON encodes v as JSON with the given status code. Encode errors are
// explicitly ignored: once the status header has been written the response is
// already committed and there is nothing useful to do with a late write error.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
