// Package pagination provides generic, secure HTTP pagination helpers for
// copy-paste backends. It supports two paging models with a single, JSON-ready
// generic envelope:
//
//   - offset/limit paging (OffsetPaginator), driven by the query parameters
//     offset, limit, page and per_page;
//   - cursor paging (CursorPaginator), driven by a base64-encoded cursor token
//     plus limit/per_page and a direction.
//
// The package uses only the Go standard library. Every value parsed from a
// client request is validated and clamped to a safe range before use, so that
// callers can never receive out-of-bounds offsets, oversized limits or
// untrusted cursor payloads.
package pagination

// Options configures the behaviour of the offset/limit and cursor paginators.
//
// All limits are enforced after parsing: clients cannot raise them by sending
// larger values. Obtain a sane starting point with Defaults() and then override
// individual fields as needed.
type Options struct {
	// DefaultLimit is the page size used when the client omits limit/per_page.
	DefaultLimit int
	// MaxLimit is the largest page size a client may request. Values above
	// this are silently clamped down. Defaults to 100.
	MaxLimit int
	// MaxOffset caps the absolute offset a client may request. A value of 0
	// means "no cap" (the application is still responsible for bounding its
	// own queries). Negative offsets are always normalised to 0.
	MaxOffset int
	// MaxCursorLen is the maximum number of bytes allowed in a cursor after
	// base64 decoding. Tokens whose decoded form exceeds this are rejected.
	// Defaults to 256.
	MaxCursorLen int
}

// Defaults returns the recommended Options: a 20-item default page, a 100-item
// hard ceiling, no offset cap and a 256-byte cursor limit.
func Defaults() Options {
	return Options{
		DefaultLimit: 20,
		MaxLimit:     100,
		MaxOffset:    0,
		MaxCursorLen: 256,
	}
}

// normalize fills in any zero/invalid fields with their defaults so that the
// rest of the package can rely on every field being within range.
func (o Options) normalize() Options {
	if o.DefaultLimit < 1 {
		o.DefaultLimit = 20
	}
	if o.MaxLimit < 1 {
		o.MaxLimit = 100
	}
	if o.DefaultLimit > o.MaxLimit {
		o.DefaultLimit = o.MaxLimit
	}
	if o.MaxOffset < 0 {
		o.MaxOffset = 0
	}
	if o.MaxCursorLen < 1 {
		o.MaxCursorLen = 256
	}
	return o
}
