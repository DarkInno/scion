package pagination

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mustReq builds a GET *http.Request for the given target. Shared by all test
// files in this package.
func mustReq(t *testing.T, target string) *http.Request {
	t.Helper()
	return httptest.NewRequest(http.MethodGet, target, nil)
}

// paginationMap unmarshals b and returns the "pagination" sub-object as a map,
// failing the test if the shape is wrong. Shared by all test files.
func paginationMap(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pm, ok := m["pagination"].(map[string]any)
	if !ok {
		t.Fatalf("pagination is not a JSON object: %v", m["pagination"])
	}
	return pm
}

func TestDefaults(t *testing.T) {
	o := Defaults()
	if o.DefaultLimit != 20 || o.MaxLimit != 100 || o.MaxOffset != 0 || o.MaxCursorLen != 256 {
		t.Fatalf("Defaults = %+v, want {20 100 0 256}", o)
	}
}

func TestParseOffsetDefaults(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items"))
	if op.Offset != 0 || op.Limit != 20 || op.Page != 1 || op.PerPage != 20 {
		t.Fatalf("defaults: got %+v, want offset=0 limit=20 page=1 per_page=20", op)
	}
}

func TestParseOffsetLimitAndOffset(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items?offset=10&limit=5"))
	if op.Offset != 10 || op.Limit != 5 || op.Page != 3 || op.PerPage != 5 {
		t.Fatalf("got %+v, want offset=10 limit=5 page=3", op)
	}
}

func TestParseOffsetPageAndPerPage(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items?page=3&per_page=10"))
	if op.Offset != 20 || op.Limit != 10 || op.Page != 3 || op.PerPage != 10 {
		t.Fatalf("got %+v, want offset=20 limit=10 page=3", op)
	}
}

func TestParseOffsetLimitOverridesPerPage(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items?per_page=50&limit=7"))
	if op.Limit != 7 {
		t.Fatalf("limit should win over per_page: got %d", op.Limit)
	}
}

func TestParseOffsetExplicitOffsetBeatsPage(t *testing.T) {
	// When both offset and page are given, the explicit offset wins and page is
	// recomputed from it.
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items?page=5&offset=30&limit=10"))
	if op.Offset != 30 {
		t.Fatalf("offset = %d, want 30 (explicit wins)", op.Offset)
	}
	if op.Page != 4 {
		t.Fatalf("page = %d, want 4 (recomputed from offset/limit)", op.Page)
	}
}

func TestParseOffsetLimitClampedToMax(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items?limit=9999"))
	if op.Limit != 100 {
		t.Fatalf("limit = %d, want 100 (clamped to MaxLimit)", op.Limit)
	}
}

func TestParseOffsetNegativeOffsetBecomesZero(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items?offset=-50&limit=10"))
	if op.Offset != 0 {
		t.Fatalf("negative offset leaked: got %d, want 0", op.Offset)
	}
	if op.Page != 1 {
		t.Fatalf("page = %d, want 1", op.Page)
	}
}

func TestParseOffsetNegativeLimitBecomesOne(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items?limit=-5"))
	if op.Limit != 1 {
		t.Fatalf("negative limit leaked: got %d, want 1", op.Limit)
	}
}

func TestParseOffsetZeroLimitBecomesOne(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items?limit=0"))
	if op.Limit != 1 {
		t.Fatalf("zero limit: got %d, want 1", op.Limit)
	}
}

func TestParseOffsetNonNumericIgnored(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	op := p.Parse(mustReq(t, "/items?offset=abc&limit=xyz&page=foo"))
	if op.Offset != 0 || op.Limit != 20 || op.Page != 1 {
		t.Fatalf("non-numeric should fall back to defaults: got %+v", op)
	}
}

func TestParseOffsetCustomOptions(t *testing.T) {
	p := NewOffsetPaginator[int](Options{DefaultLimit: 5, MaxLimit: 10})
	op := p.Parse(mustReq(t, "/items"))
	if op.Limit != 5 {
		t.Fatalf("custom default limit = %d, want 5", op.Limit)
	}
	op2 := p.Parse(mustReq(t, "/items?limit=50"))
	if op2.Limit != 10 {
		t.Fatalf("custom max limit clamp = %d, want 10", op2.Limit)
	}
}

func TestParseOffsetMaxOffsetCap(t *testing.T) {
	p := NewOffsetPaginator[int](Options{DefaultLimit: 10, MaxLimit: 100, MaxOffset: 1000})
	op := p.Parse(mustReq(t, "/items?offset=99999"))
	if op.Offset != 1000 {
		t.Fatalf("offset = %d, want 1000 (MaxOffset cap)", op.Offset)
	}
}

func TestOffsetPaginateMiddlePage(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	data := []int{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	res := p.Paginate(data, 25, OffsetParams{Offset: 10, Limit: 10, Page: 2, PerPage: 10})
	if *res.Pagination.Total != 25 {
		t.Fatalf("total = %d, want 25", *res.Pagination.Total)
	}
	if *res.Pagination.Page != 2 {
		t.Fatalf("page = %d, want 2", *res.Pagination.Page)
	}
	if *res.Pagination.PerPage != 10 {
		t.Fatalf("per_page = %d, want 10", *res.Pagination.PerPage)
	}
	if *res.Pagination.TotalPages != 3 {
		t.Fatalf("total_pages = %d, want 3", *res.Pagination.TotalPages)
	}
	if !res.Pagination.HasNext {
		t.Fatal("HasNext should be true on page 2 of 3")
	}
	if !res.Pagination.HasPrev {
		t.Fatal("HasPrev should be true on page 2 of 3")
	}
}

func TestOffsetPaginateFirstPage(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	res := p.Paginate([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 25, OffsetParams{Offset: 0, Limit: 10, Page: 1, PerPage: 10})
	if res.Pagination.HasPrev {
		t.Fatal("HasPrev should be false on page 1")
	}
	if !res.Pagination.HasNext {
		t.Fatal("HasNext should be true on page 1 of 3")
	}
}

func TestOffsetPaginateLastPage(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	res := p.Paginate([]int{21, 22, 23, 24, 25}, 25, OffsetParams{Offset: 20, Limit: 10, Page: 3, PerPage: 10})
	if res.Pagination.HasNext {
		t.Fatal("HasNext should be false on last page")
	}
	if !res.Pagination.HasPrev {
		t.Fatal("HasPrev should be true on last page")
	}
}

func TestOffsetPaginateZeroTotal(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	res := p.Paginate([]int{}, 0, OffsetParams{Offset: 0, Limit: 20, Page: 1, PerPage: 20})
	if *res.Pagination.TotalPages != 0 {
		t.Fatalf("total_pages = %d, want 0", *res.Pagination.TotalPages)
	}
	if res.Pagination.HasNext || res.Pagination.HasPrev {
		t.Fatal("no next/prev with zero total")
	}
}

func TestOffsetPaginateExactDivision(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	res := p.Paginate([]int{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, 20, OffsetParams{Offset: 10, Limit: 10, Page: 2, PerPage: 10})
	if *res.Pagination.TotalPages != 2 {
		t.Fatalf("total_pages = %d, want 2", *res.Pagination.TotalPages)
	}
	if res.Pagination.HasNext {
		t.Fatal("HasNext should be false on final page with exact division")
	}
}

func TestOffsetPaginateShortPageHasNoNext(t *testing.T) {
	// A page shorter than the limit is the final page by definition: never
	// advertise a next page that cannot be served, even with an inflated total.
	p := NewOffsetPaginator[int](Defaults())
	res := p.Paginate([]int{1, 2, 3}, 25, OffsetParams{Offset: 0, Limit: 10, Page: 1, PerPage: 10})
	if res.Pagination.HasNext {
		t.Fatal("HasNext should be false when the page is shorter than the limit")
	}
}

func TestOffsetPaginateRecomputesPage(t *testing.T) {
	// When Page is left at zero, Paginate recomputes it from Offset/Limit.
	p := NewOffsetPaginator[int](Defaults())
	res := p.Paginate([]int{1, 2}, 50, OffsetParams{Offset: 40, Limit: 10, PerPage: 10})
	if *res.Pagination.Page != 5 {
		t.Fatalf("recomputed page = %d, want 5", *res.Pagination.Page)
	}
}

func TestOffsetResultJSONShape(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	res := p.Paginate([]int{1, 2, 3}, 25, OffsetParams{Offset: 0, Limit: 10, Page: 1, PerPage: 10})
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pm := paginationMap(t, b)
	for _, k := range []string{"total", "page", "per_page", "total_pages", "has_next", "has_prev"} {
		if _, ok := pm[k]; !ok {
			t.Fatalf("offset JSON missing key %q in pagination: %v", k, pm)
		}
	}
	// Cursor-mode keys must NOT appear in an offset response.
	for _, k := range []string{"next_cursor", "prev_cursor"} {
		if _, ok := pm[k]; ok {
			t.Fatalf("offset JSON should not contain cursor key %q: %v", k, pm)
		}
	}
	if _, ok := pm["total"]; !ok {
		t.Fatal("data key missing")
	}
}

func TestOffsetResultNilDataRendersEmptyArray(t *testing.T) {
	p := NewOffsetPaginator[int](Defaults())
	res := p.Paginate(nil, 0, OffsetParams{Offset: 0, Limit: 20, Page: 1, PerPage: 20})
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data, ok := m["data"].([]any)
	if !ok {
		t.Fatalf("data should be a JSON array, got %T", m["data"])
	}
	if len(data) != 0 {
		t.Fatalf("nil data should render as [], got %v", data)
	}
}

func TestOffsetResultWriteJSON(t *testing.T) {
	p := NewOffsetPaginator[string](Defaults())
	res := p.Paginate([]string{"a", "b"}, 5, OffsetParams{Offset: 0, Limit: 2, Page: 1, PerPage: 2})
	rec := httptest.NewRecorder()
	res.WriteJSON(rec)
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	pm := paginationMap(t, rec.Body.Bytes())
	if pm["has_next"] != true {
		t.Fatalf("has_next = %v, want true", pm["has_next"])
	}
}

func TestOffsetPaginatorGenericWithStringType(t *testing.T) {
	p := NewOffsetPaginator[string](Defaults())
	op := p.Parse(mustReq(t, "/items?limit=2"))
	res := p.Paginate([]string{"alpha", "beta"}, 5, op)
	if len(res.Data) != 2 || res.Data[0] != "alpha" {
		t.Fatalf("generic string data mismatch: %+v", res.Data)
	}
}

func TestOffsetPaginatorGenericWithStructType(t *testing.T) {
	type item struct {
		ID int `json:"id"`
	}
	p := NewOffsetPaginator[item](Defaults())
	res := p.Paginate([]item{{ID: 1}, {ID: 2}}, 2, OffsetParams{Offset: 0, Limit: 10, Page: 1, PerPage: 10})
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data, ok := m["data"].([]any)
	if !ok || len(data) != 2 {
		t.Fatalf("struct data mismatch: %v", m["data"])
	}
}
