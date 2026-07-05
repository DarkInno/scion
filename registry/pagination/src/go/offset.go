package pagination

import (
	"net/http"
	"net/url"
	"strconv"
)

// OffsetParams holds the validated offset/limit pagination parameters parsed
// from a request. All fields are guaranteed to be within their allowed bounds.
type OffsetParams struct {
	// Offset is the zero-based record offset, always >= 0.
	Offset int
	// Limit is the page size, clamped to [1, MaxLimit].
	Limit int
	// Page is the 1-based page number derived from Offset/Limit (or supplied
	// directly), always >= 1.
	Page int
	// PerPage is an alias for Limit, kept for symmetry with the per_page
	// query parameter.
	PerPage int
}

// OffsetPaginator parses offset/limit query parameters and builds offset-mode
// PaginatedResult values. It is generic over the data element type.
type OffsetPaginator[T any] struct {
	opts Options
}

// NewOffsetPaginator returns an OffsetPaginator configured with opts (which are
// normalised to safe defaults).
func NewOffsetPaginator[T any](opts Options) *OffsetPaginator[T] {
	return &OffsetPaginator[T]{opts: opts.normalize()}
}

// Options returns the (normalised) options used by the paginator.
func (p *OffsetPaginator[T]) Options() Options { return p.opts }

// Parse extracts and validates offset/limit parameters from r. It never fails:
// any missing, non-numeric or out-of-range value is corrected to a safe
// default, so the returned OffsetParams is always usable.
func (p *OffsetPaginator[T]) Parse(r *http.Request) OffsetParams {
	return parseOffsetParams(r.URL.Query(), p.opts)
}

// Paginate builds an offset-mode PaginatedResult from a page of data and the
// total record count. HasNext/HasPrev are derived from the offset, the page
// size and the total.
func (p *OffsetPaginator[T]) Paginate(data []T, total int64, params OffsetParams) PaginatedResult[T] {
	limit := params.Limit
	if limit < 1 {
		limit = p.opts.DefaultLimit
	}
	page := params.Page
	if page < 1 {
		if limit > 0 {
			page = (params.Offset / limit) + 1
		} else {
			page = 1
		}
	}

	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	hasNext := false
	if limit > 0 && int64(params.Offset)+int64(limit) < total {
		hasNext = true
	}
	// A page shorter than the limit is the final page by definition, even when
	// the caller passed an inflated total: never advertise a next page that
	// cannot be served.
	if limit > 0 && len(data) < limit {
		hasNext = false
	}
	hasPrev := params.Offset > 0

	t := total
	pg := page
	pp := limit
	tp := totalPages
	return PaginatedResult[T]{
		Data: data,
		Pagination: PaginationMeta{
			Total:      &t,
			Page:       &pg,
			PerPage:    &pp,
			TotalPages: &tp,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}
}

// parseOffsetParams resolves offset/limit from a query value set.
//
// Recognised parameters (last wins for limit):
//
//	per_page  integer page size (alias for limit)
//	limit     integer page size
//	offset    zero-based record offset
//	page      1-based page number; when supplied without an explicit offset,
//	          offset is derived as (page-1)*limit
//
// Negative offsets become 0; limits are clamped to [1, MaxLimit]. Non-numeric
// and absurdly long values are ignored, leaving the safe default in place.
func parseOffsetParams(q url.Values, opts Options) OffsetParams {
	opts = opts.normalize()

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

	offset := 0
	explicitOffset := false
	if s := q.Get("offset"); s != "" {
		if v, ok := atoiSafe(s); ok {
			offset = v
			explicitOffset = true
		}
	}
	if offset < 0 {
		offset = 0
	}
	if opts.MaxOffset > 0 && offset > opts.MaxOffset {
		offset = opts.MaxOffset
	}

	page := 0
	if s := q.Get("page"); s != "" {
		if v, ok := atoiSafe(s); ok {
			page = v
		}
	}
	if page < 0 {
		page = 0
	}
	if page >= 1 && !explicitOffset {
		offset = (page - 1) * limit
		if offset < 0 {
			offset = 0
		}
	}
	// Derive page from the (possibly explicit) offset so the reported page is
	// always consistent with the offset/limit actually used. When the client
	// supplies both page and an explicit offset, the explicit offset wins.
	if explicitOffset || page < 1 {
		if limit > 0 {
			page = (offset / limit) + 1
		} else {
			page = 1
		}
	}

	return OffsetParams{
		Offset:  offset,
		Limit:   limit,
		Page:    page,
		PerPage: limit,
	}
}

// atoiSafe parses s as an int but only when it is short enough to plausibly be a
// valid integer. Overlong values (which can only be overflow or garbage) are
// rejected, guarding against pathological inputs such as multi-megabyte query
// parameters.
func atoiSafe(s string) (int, bool) {
	const maxIntLen = 20 // len("-9223372036854775808")
	if len(s) == 0 || len(s) > maxIntLen {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}
