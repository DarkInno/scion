package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// RegisterRoutes registers auth endpoints on the given mux.
// Use RoutePrefix to customize the path prefix.
//
// WARNING: http.ServeMux automatically redirects /prefix to /prefix/
// with a 301 status. Many HTTP clients (including Go's http.Client)
// will change POST to GET on a 301 redirect. Register both forms
// or ensure clients always use the exact registered path.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	prefix := RoutePrefix
	mux.HandleFunc("POST "+prefix+"/register", h.Register)
	mux.HandleFunc("POST "+prefix+"/login", h.Login)
	mux.Handle("GET "+prefix+"/me", h.Middleware(http.HandlerFunc(h.Me)))
}

// RoutePrefix is the default API path prefix for auth routes.
// Override this before calling RegisterRoutes if you need a different prefix.
var RoutePrefix = "/api/v1/auth"

// NormalizeEmail lowercases and trims an email address.
// Use this before storing or looking up emails to ensure case-insensitive matching.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
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
	if path == "/" {
		return "/", nil
	}
	return strings.TrimSuffix(path, "/"), nil
}
