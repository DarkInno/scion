package middleware

// contextKey is an unexported type for context keys defined in this package.
// Using an unexported type prevents collisions with keys from other packages.
type contextKey int

const (
	requestIDKey contextKey = iota
	clientIPKey
	traceIDKey
	spanIDKey
	traceParentKey
	baggageKey
)
