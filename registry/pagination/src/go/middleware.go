package pagination

import (
	"context"
	"net/http"
)

// contextKey is an unexported type so external packages cannot forge context
// values.
type contextKey int

const (
	optionsKey contextKey = iota
	offsetParamsKey
	cursorParamsKey
	cursorErrKey
)

// Middleware returns an http.Handler middleware (signature
// func(http.Handler) http.Handler) that parses both offset/limit and cursor
// pagination parameters from the request query string and stores them in the
// request context for downstream handlers.
//
// Offset parameters are always parsed and clamped to safe bounds. Cursor
// parameters are parsed when a "cursor" query parameter is present; a
// malformed cursor is stored as an error retrievable via CursorErrorFromContext
// rather than aborting the request, so the handler can decide how to respond.
func Middleware(opts Options) func(http.Handler) http.Handler {
	opts = opts.normalize()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, optionsKey, opts)
			ctx = context.WithValue(ctx, offsetParamsKey, parseOffsetParams(r.URL.Query(), opts))
			cp, err := parseCursorParams(r.URL.Query(), opts)
			ctx = context.WithValue(ctx, cursorParamsKey, cp)
			if err != nil {
				ctx = context.WithValue(ctx, cursorErrKey, err)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionsFromContext returns the Options stored by the middleware.
func OptionsFromContext(ctx context.Context) (Options, bool) {
	o, ok := ctx.Value(optionsKey).(Options)
	return o, ok
}

// OffsetFromContext returns the OffsetParams stored by the middleware.
func OffsetFromContext(ctx context.Context) (OffsetParams, bool) {
	p, ok := ctx.Value(offsetParamsKey).(OffsetParams)
	return p, ok
}

// CursorFromContext returns the CursorParams stored by the middleware.
func CursorFromContext(ctx context.Context) (CursorParams, bool) {
	p, ok := ctx.Value(cursorParamsKey).(CursorParams)
	return p, ok
}

// CursorErrorFromContext returns the error produced while parsing the cursor,
// if any. Handlers in cursor mode should check this and reject the request
// (e.g. with 400 Bad Request) when it is non-nil, rather than trusting the
// client-supplied token.
func CursorErrorFromContext(ctx context.Context) error {
	if err, ok := ctx.Value(cursorErrKey).(error); ok {
		return err
	}
	return nil
}
