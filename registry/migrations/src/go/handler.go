package migrations

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// StatusHandler returns an HTTP handler that reports migration status as JSON.
// The handler recovers from panics and never exposes request headers or bodies.
func (m *Migrator) StatusHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		statuses, err := m.Status(r.Context(), db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "migration status unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, statuses)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
