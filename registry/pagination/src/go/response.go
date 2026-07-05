package pagination

import (
	"encoding/json"
	"net/http"
)

// PaginationMeta carries the pagination metadata for a result. The offset/limit
// fields (Total, Page, PerPage, TotalPages) are populated for offset mode; the
// cursor fields (NextCursor, PrevCursor) are populated for cursor mode. Unused
// fields are nil and omitted from the JSON output. HasNext and HasPrev are
// always present so clients can branch uniformly.
type PaginationMeta struct {
	// Offset/limit mode fields.
	Total      *int64 `json:"total,omitempty"`
	Page       *int   `json:"page,omitempty"`
	PerPage    *int   `json:"per_page,omitempty"`
	TotalPages *int   `json:"total_pages,omitempty"`
	// Cursor mode fields.
	NextCursor *string `json:"next_cursor,omitempty"`
	PrevCursor *string `json:"prev_cursor,omitempty"`
	// Common flags, always serialized.
	HasNext bool `json:"has_next"`
	HasPrev bool `json:"has_prev"`
}

// PaginatedResult is the generic, JSON-friendly paginated envelope returned by
// both the offset/limit and cursor paginators. It is parameterised over the
// data element type T (Go 1.22 generics).
type PaginatedResult[T any] struct {
	Data       []T            `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// MarshalJSON ensures a nil Data slice is rendered as "[]" rather than "null",
// matching the documented {"data": [...]} contract, and delegates everything
// else to the default encoding.
func (r PaginatedResult[T]) MarshalJSON() ([]byte, error) {
	data := r.Data
	if data == nil {
		data = []T{}
	}
	return json.Marshal(struct {
		Data       []T            `json:"data"`
		Pagination PaginationMeta `json:"pagination"`
	}{
		Data:       data,
		Pagination: r.Pagination,
	})
}

// WriteJSON serialises the result as JSON to w. Any encoding error is
// deliberately ignored: an HTTP response cannot meaningfully surface a
// mid-stream JSON failure to the caller, and the response status/headers may
// already be committed.
func (r PaginatedResult[T]) WriteJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(r)
}
