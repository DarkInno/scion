package middleware

import "net/http"

// ChainBuilder composes multiple middlewares into a single chain.
// Execution order: left to right (outermost to innermost).
//
//	Chain(m1, m2, m3).Then(h)  =>  m1(m2(m3(h)))
//
// Request flows: m1 -> m2 -> m3 -> h
// Response flows: h -> m3 -> m2 -> m1
type ChainBuilder struct {
	middlewares []func(http.Handler) http.Handler
}

// Chain creates a new ChainBuilder from the given middlewares.
// A defensive copy is made to prevent external mutation.
func Chain(middlewares ...func(http.Handler) http.Handler) *ChainBuilder {
	mws := make([]func(http.Handler) http.Handler, len(middlewares))
	copy(mws, middlewares)
	return &ChainBuilder{middlewares: mws}
}

// Then applies the middleware chain to the final handler.
// Wraps from right to left so request flows left to right.
// If final is nil, http.NotFoundHandler is used.
// If any middleware returns nil, it degrades to http.NotFoundHandler.
func (c *ChainBuilder) Then(final http.Handler) http.Handler {
	if final == nil {
		final = http.NotFoundHandler()
	}
	h := final
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		if c.middlewares[i] == nil {
			// Nil middleware: degrade to passthrough to avoid panic.
			continue
		}
		h = c.middlewares[i](h)
		if h == nil {
			// Middleware returned nil: degrade to NotFoundHandler.
			h = http.NotFoundHandler()
		}
	}
	return h
}

// ThenFunc is a convenience method that wraps a handler function.
func (c *ChainBuilder) ThenFunc(final func(http.ResponseWriter, *http.Request)) http.Handler {
	if final == nil {
		return c.Then(nil)
	}
	return c.Then(http.HandlerFunc(final))
}

// Append returns a new ChainBuilder with additional middlewares appended.
// The original chain is not modified (immutable).
func (c *ChainBuilder) Append(mws ...func(http.Handler) http.Handler) *ChainBuilder {
	combined := make([]func(http.Handler) http.Handler, 0, len(c.middlewares)+len(mws))
	combined = append(combined, c.middlewares...)
	combined = append(combined, mws...)
	return &ChainBuilder{middlewares: combined}
}
