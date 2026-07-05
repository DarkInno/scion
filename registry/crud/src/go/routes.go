package crud

import (
	"fmt"
	"net/http"
	"strings"
)

// RegisterRoutes registers CRUD endpoints on the given mux.
// basePath should be a clean path like "/api/v1/products" (no trailing slash).
//
// WARNING: http.ServeMux automatically redirects /prefix to /prefix/
// with a 301 status. Many HTTP clients (including Go's http.Client)
// will change POST to GET on a 301 redirect. Always pass paths
// without trailing slashes to this function.
func (h *Handler[T]) RegisterRoutes(mux *http.ServeMux, basePath string) error {
	basePath, err := ValidateRoutePrefix(basePath)
	if err != nil {
		return fmt.Errorf("crud: %w", err)
	}

	mux.HandleFunc("POST "+basePath, h.Create)
	mux.HandleFunc("GET "+basePath, h.List)
	mux.HandleFunc("GET "+basePath+"/{id}", h.Get)
	mux.HandleFunc("PUT "+basePath+"/{id}", h.Update)
	mux.HandleFunc("DELETE "+basePath+"/{id}", h.Delete)
	return nil
}

// ValidateRoutePrefix checks that a path prefix is well-formed.
// Returns a cleaned prefix (leading "/", no trailing "/") or an error.
func ValidateRoutePrefix(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("route prefix must not be empty")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimSuffix(path, "/"), nil
}
