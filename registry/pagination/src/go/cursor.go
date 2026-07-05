package pagination

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// ErrInvalidCursor is returned when a cursor token is empty, not valid base64,
// too long, or contains CR/LF bytes after decoding. Callers must treat a cursor
// that fails to decode as absent rather than trusting the raw client input.
var ErrInvalidCursor = errors.New("pagination: invalid cursor")

// CursorDirection indicates which way a client is paging.
type CursorDirection string

const (
	// CursorNext pages forward (the common case).
	CursorNext CursorDirection = "next"
	// CursorPrev pages backward.
	CursorPrev CursorDirection = "prev"
)

// CursorParams holds the validated cursor pagination parameters parsed from a
// request. Cursor is the decoded, validated opaque position marker; it is empty
// when the client supplied no cursor (i.e. the first page).
type CursorParams struct {
	// Cursor is the decoded cursor value supplied by the client. Empty means
	// "start from the beginning".
	Cursor string
	// Limit is the page size, clamped to [1, MaxLimit].
	Limit int
	// Direction is the paging direction requested by the client.
	Direction CursorDirection
}

// CursorPaginator parses cursor query parameters, encodes/decodes opaque cursor
// tokens and builds cursor-mode PaginatedResult values. It is generic over the
// data element type.
type CursorPaginator[T any] struct {
	opts Options
}

// NewCursorPaginator returns a CursorPaginator configured with opts.
func NewCursorPaginator[T any](opts Options) *CursorPaginator[T] {
	return &CursorPaginator[T]{opts: opts.normalize()}
}

// Options returns the (normalised) options used by the paginator.
func (p *CursorPaginator[T]) Options() Options { return p.opts }

// Parse extracts and validates cursor parameters from r. A missing cursor is
// not an error (it simply means "first page"); a malformed cursor is reported
// via ErrInvalidCursor.
func (p *CursorPaginator[T]) Parse(r *http.Request) (CursorParams, error) {
	return parseCursorParams(r.URL.Query(), p.opts)
}

// Paginate builds a cursor-mode PaginatedResult. nextCursor/prevCursor are the
// raw (un-encoded) opaque position markers the application wishes to hand back
// to the client; they are base64-encoded here so callers never handle the wire
// format directly. When hasNext/hasPrev is false the corresponding cursor is
// omitted from the response.
func (p *CursorPaginator[T]) Paginate(data []T, nextCursor, prevCursor string, hasNext, hasPrev bool) PaginatedResult[T] {
	meta := PaginationMeta{
		HasNext: hasNext,
		HasPrev: hasPrev,
	}
	if hasNext && nextCursor != "" {
		enc := EncodeCursor(nextCursor)
		meta.NextCursor = &enc
	}
	if hasPrev && prevCursor != "" {
		enc := EncodeCursor(prevCursor)
		meta.PrevCursor = &enc
	}
	return PaginatedResult[T]{Data: data, Pagination: meta}
}

// EncodeCursor base64-encodes a raw cursor value into an opaque, URL-safe
// token. The encoding is reversible by DecodeCursor; the value itself is not
// encrypted or authenticated, so cursors must never carry trusted or
// security-sensitive data.
func EncodeCursor(raw string) string {
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor base64-decodes and validates a client-supplied cursor token.
// Validation enforces:
//   - a hard cap on the encoded length (to bound decoding cost and memory);
//   - a hard cap on the decoded length (maxLen, default 256);
//   - absence of CR/LF bytes in the decoded value (to prevent log/header
//     injection downstream).
//
// The decoded string is returned only when every check passes; otherwise
// ErrInvalidCursor is returned and the caller must treat the cursor as absent.
// The client's raw token is never trusted directly.
func DecodeCursor(token string, maxLen int) (string, error) {
	if maxLen < 1 {
		maxLen = 256
	}
	// Bound the encoded size we are willing to decode. base64 expands by ~4/3,
	// so 4*maxLen plus a little padding is a generous ceiling; anything larger
	// is rejected before any allocation happens.
	maxEncoded := 4*maxLen + 8
	if len(token) == 0 {
		return "", ErrInvalidCursor
	}
	if len(token) > maxEncoded {
		return "", ErrInvalidCursor
	}
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		// Tolerate tokens whose padding was stripped by an intermediary.
		decoded, err = base64.RawURLEncoding.DecodeString(token)
		if err != nil {
			return "", ErrInvalidCursor
		}
	}
	if len(decoded) > maxLen {
		return "", ErrInvalidCursor
	}
	// Reject CR/LF to prevent header/log injection downstream.
	if strings.ContainsRune(string(decoded), '\r') || strings.ContainsRune(string(decoded), '\n') {
		return "", ErrInvalidCursor
	}
	return string(decoded), nil
}

// parseCursorParams resolves cursor parameters from a query value set.
//
// Recognised parameters:
//
//	cursor     opaque base64 cursor token (enables cursor mode); missing/empty
//	           means "first page"
//	limit      page size, clamped to [1, MaxLimit]
//	per_page   alias for limit
//	direction  "next" (default) or "prev"
func parseCursorParams(q url.Values, opts Options) (CursorParams, error) {
	opts = opts.normalize()
	params := CursorParams{
		Limit:     opts.DefaultLimit,
		Direction: CursorNext,
	}

	limit := opts.DefaultLimit
	if s := q.Get("per_page"); s != "" {
		if v, ok := atoiSafe(s); ok {
			limit = v
		}
	}
	if s := q.Get("limit"); s != "" {
		if v, ok := atoiSafe(s); ok {
			limit = v
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > opts.MaxLimit {
		limit = opts.MaxLimit
	}
	params.Limit = limit

	if c := q.Get("cursor"); c != "" {
		decoded, err := DecodeCursor(c, opts.MaxCursorLen)
		if err != nil {
			return params, err
		}
		params.Cursor = decoded
	}

	switch CursorDirection(strings.ToLower(q.Get("direction"))) {
	case CursorPrev:
		params.Direction = CursorPrev
	default:
		params.Direction = CursorNext
	}
	return params, nil
}
