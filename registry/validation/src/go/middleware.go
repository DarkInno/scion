package validation

import (
	"encoding/json"
	"net/http"
)

// Schema is the minimal interface a validation schema implements. Both *Builder
// and the *Field returned by a fluent chain satisfy it, so Middleware can be
// given either form:
//
//	// multi-line, keeping the builder reference:
//	schema := validation.New()
//	schema.Field("email").Required().Email()
//	mw := validation.Middleware(schema)
//
//	// or directly from the chain tail:
//	mw := validation.Middleware(
//	    validation.New().Field("email").Required().Email(),
//	)
type Schema interface {
	ValidateRequest(r *http.Request) *ValidationError
}

// asBuilder extracts the underlying *Builder from a Schema so that option
// overrides can be applied. It returns nil for unknown implementations.
func asBuilder(s Schema) *Builder {
	switch v := s.(type) {
	case *Builder:
		return v
	case *Field:
		return v.builder
	}
	return nil
}

// Middleware returns an http.Handler middleware of the standard signature
// func(http.Handler) http.Handler. Incoming requests are validated against
// schema; on validation failure the middleware short-circuits and responds with
// HTTP 422 Unprocessable Entity and a JSON body of the form:
//
//	{"errors":{"field":["message", ...]}}
//
// opts may be used to override the schema's options for this middleware only
// (for example to validate a different Source). The middleware never panics:
// any panic raised while validating (including from a Custom rule) is recovered
// and converted into a 422 response with an "_server" error, ensuring the
// module's "no panic" guarantee holds even with untrusted rule functions.
func Middleware(schema Schema, opts ...Option) func(http.Handler) http.Handler {
	effective := asBuilder(schema)
	if effective != nil && len(opts) > 0 {
		effective = effective.withOptions(opts)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if effective == nil {
				next.ServeHTTP(w, r)
				return
			}
			verr := validateSafely(effective, r)
			if verr.HasErrors() {
				_ = verr.WriteJSON(w, http.StatusUnprocessableEntity)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// validateSafely runs ValidateRequest, recovering from any panic so that the
// module never propagates a panic to the caller.
func validateSafely(schema *Builder, r *http.Request) (ve *ValidationError) {
	ve = NewValidationError()
	defer func() {
		if rec := recover(); rec != nil {
			ve = NewValidationError()
			ve.Add("_server", "internal validation error")
		}
	}()
	ve = schema.ValidateRequest(r)
	return ve
}

// WriteJSON writes the validation error as a JSON response with the given HTTP
// status code. It is a convenience for handlers that validate manually (without
// the middleware) and need to emit a structured error response.
func (ve *ValidationError) WriteJSON(w http.ResponseWriter, status int) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(ve)
}
