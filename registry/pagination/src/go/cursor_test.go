package pagination

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	cases := []string{
		"id:42",
		"2024-01-01T00:00:00Z|12345",
		"opaque-token-with-spaces and symbols !@#$%",
		"x",
	}
	for _, raw := range cases {
		tok := EncodeCursor(raw)
		got, err := DecodeCursor(tok, 256)
		if err != nil {
			t.Fatalf("DecodeCursor(%q) error: %v", raw, err)
		}
		if got != raw {
			t.Fatalf("round trip mismatch: got %q want %q", got, raw)
		}
	}
}

func TestDecodeCursorEmpty(t *testing.T) {
	if _, err := DecodeCursor("", 256); err != ErrInvalidCursor {
		t.Fatalf("empty cursor err = %v, want ErrInvalidCursor", err)
	}
}

func TestDecodeCursorInvalidBase64(t *testing.T) {
	cases := []string{
		"!!!not-base64!!!",
		"@#$%",
		"abc def", // space is not valid base64
		"====",
		"????",
	}
	for _, tok := range cases {
		if _, err := DecodeCursor(tok, 256); err != ErrInvalidCursor {
			t.Fatalf("DecodeCursor(%q) err = %v, want ErrInvalidCursor", tok, err)
		}
	}
}

func TestDecodeCursorTooLongDecoded(t *testing.T) {
	raw := strings.Repeat("a", 300) // exceeds the 256-byte limit
	tok := EncodeCursor(raw)
	if _, err := DecodeCursor(tok, 256); err != ErrInvalidCursor {
		t.Fatalf("over-length decoded cursor should be rejected, got err = %v", err)
	}
}

func TestDecodeCursorRespectsCustomMaxLen(t *testing.T) {
	raw := strings.Repeat("z", 50)
	tok := EncodeCursor(raw)
	// 50 bytes is fine under a 100-byte limit but rejected under a 10-byte one.
	if _, err := DecodeCursor(tok, 100); err != nil {
		t.Fatalf("50-byte cursor under 100 limit should pass, got %v", err)
	}
	if _, err := DecodeCursor(tok, 10); err != ErrInvalidCursor {
		t.Fatalf("50-byte cursor under 10 limit should be rejected, got %v", err)
	}
}

func TestDecodeCursorContainsCRLF(t *testing.T) {
	cases := []string{
		"abc\r\ndef",
		"abc\rdef",
		"abc\ndef",
	}
	for _, raw := range cases {
		tok := EncodeCursor(raw)
		if _, err := DecodeCursor(tok, 256); err != ErrInvalidCursor {
			t.Fatalf("cursor with CRLF should be rejected for %q, got %v", raw, err)
		}
	}
}

func TestDecodeCursorOversizedEncoded(t *testing.T) {
	// An encoded token far larger than the allowed ceiling is rejected before
	// any decoding allocation happens.
	tok := strings.Repeat("A", 5000)
	if _, err := DecodeCursor(tok, 256); err != ErrInvalidCursor {
		t.Fatalf("oversized encoded cursor should be rejected, got %v", err)
	}
}

func TestCursorParseDefaults(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	cp, err := p.Parse(mustReq(t, "/items"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cp.Cursor != "" {
		t.Fatalf("default cursor should be empty, got %q", cp.Cursor)
	}
	if cp.Limit != 20 {
		t.Fatalf("default limit = %d, want 20", cp.Limit)
	}
	if cp.Direction != CursorNext {
		t.Fatalf("default direction = %q, want next", cp.Direction)
	}
}

func TestCursorParseWithCursor(t *testing.T) {
	tok := EncodeCursor("id:42")
	p := NewCursorPaginator[int](Defaults())
	cp, err := p.Parse(mustReq(t, "/items?cursor="+tok+"&limit=5"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cp.Cursor != "id:42" {
		t.Fatalf("cursor = %q, want id:42", cp.Cursor)
	}
	if cp.Limit != 5 {
		t.Fatalf("limit = %d, want 5", cp.Limit)
	}
}

func TestCursorParseInvalidCursorError(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	cp, err := p.Parse(mustReq(t, "/items?cursor=!!!bad"))
	if err != ErrInvalidCursor {
		t.Fatalf("err = %v, want ErrInvalidCursor", err)
	}
	// On error the cursor must be empty: never trust the raw client input.
	if cp.Cursor != "" {
		t.Fatalf("cursor should be empty on error, got %q", cp.Cursor)
	}
}

func TestCursorParseLimitClamped(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	cp, _ := p.Parse(mustReq(t, "/items?limit=9999"))
	if cp.Limit != 100 {
		t.Fatalf("limit = %d, want 100 (clamped)", cp.Limit)
	}
}

func TestCursorParseNegativeLimit(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	cp, _ := p.Parse(mustReq(t, "/items?limit=-5"))
	if cp.Limit != 1 {
		t.Fatalf("negative limit = %d, want 1", cp.Limit)
	}
}

func TestCursorParseDirection(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	cp, _ := p.Parse(mustReq(t, "/items?direction=prev"))
	if cp.Direction != CursorPrev {
		t.Fatalf("direction = %q, want prev", cp.Direction)
	}
	cp2, _ := p.Parse(mustReq(t, "/items?direction=next"))
	if cp2.Direction != CursorNext {
		t.Fatalf("direction = %q, want next", cp2.Direction)
	}
	// Unknown direction falls back to "next" rather than erroring.
	cp3, _ := p.Parse(mustReq(t, "/items?direction=sideways"))
	if cp3.Direction != CursorNext {
		t.Fatalf("unknown direction = %q, want next (default)", cp3.Direction)
	}
}

func TestCursorPaginateWithNextAndPrev(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	res := p.Paginate([]int{1, 2, 3}, "next-id", "prev-id", true, true)
	if res.Pagination.NextCursor == nil {
		t.Fatal("NextCursor should be set")
	}
	if *res.Pagination.NextCursor != EncodeCursor("next-id") {
		t.Fatalf("NextCursor = %q, want %q", *res.Pagination.NextCursor, EncodeCursor("next-id"))
	}
	if res.Pagination.PrevCursor == nil {
		t.Fatal("PrevCursor should be set")
	}
	if *res.Pagination.PrevCursor != EncodeCursor("prev-id") {
		t.Fatalf("PrevCursor = %q, want %q", *res.Pagination.PrevCursor, EncodeCursor("prev-id"))
	}
	if !res.Pagination.HasNext || !res.Pagination.HasPrev {
		t.Fatal("HasNext and HasPrev should be true")
	}
}

func TestCursorPaginateNoNextOmitsCursor(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	res := p.Paginate([]int{1, 2, 3}, "next-id", "prev-id", false, true)
	if res.Pagination.NextCursor != nil {
		t.Fatalf("NextCursor should be nil when hasNext is false, got %v", res.Pagination.NextCursor)
	}
	if res.Pagination.PrevCursor == nil {
		t.Fatal("PrevCursor should still be set")
	}
	if res.Pagination.HasNext {
		t.Fatal("HasNext should be false")
	}
}

func TestCursorPaginateNoCursorsWhenNoMore(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	res := p.Paginate([]int{1}, "", "", false, false)
	if res.Pagination.NextCursor != nil || res.Pagination.PrevCursor != nil {
		t.Fatalf("no cursors expected, got next=%v prev=%v", res.Pagination.NextCursor, res.Pagination.PrevCursor)
	}
}

func TestCursorResultJSONShape(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	res := p.Paginate([]int{1, 2, 3}, "next-id", "", true, false)
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pm := paginationMap(t, b)
	for _, k := range []string{"next_cursor", "has_next", "has_prev"} {
		if _, ok := pm[k]; !ok {
			t.Fatalf("cursor JSON missing key %q: %v", k, pm)
		}
	}
	// Offset-mode keys must NOT appear in a cursor response.
	for _, k := range []string{"total", "page", "per_page", "total_pages"} {
		if _, ok := pm[k]; ok {
			t.Fatalf("cursor JSON should not contain offset key %q: %v", k, pm)
		}
	}
	// prev_cursor is omitted when hasPrev is false.
	if _, ok := pm["prev_cursor"]; ok {
		t.Fatalf("prev_cursor should be omitted when hasPrev is false: %v", pm)
	}
}

func TestCursorRoundTripThroughPaginate(t *testing.T) {
	// Build a result, serialize to JSON, then decode the next_cursor back to the
	// original raw value, proving the full encode/decode pipeline is consistent.
	p := NewCursorPaginator[string](Defaults())
	res := p.Paginate([]string{"a", "b"}, "last-seen-id", "", true, false)
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pm := paginationMap(t, b)
	enc, ok := pm["next_cursor"].(string)
	if !ok {
		t.Fatalf("next_cursor not a string: %v", pm["next_cursor"])
	}
	got, err := DecodeCursor(enc, 256)
	if err != nil {
		t.Fatalf("DecodeCursor: %v", err)
	}
	if got != "last-seen-id" {
		t.Fatalf("round trip = %q, want last-seen-id", got)
	}
}

func TestCursorResultWriteJSON(t *testing.T) {
	p := NewCursorPaginator[int](Defaults())
	res := p.Paginate([]int{1}, "n", "", true, false)
	rec := httptest.NewRecorder()
	res.WriteJSON(rec)
	pm := paginationMap(t, rec.Body.Bytes())
	if pm["has_next"] != true {
		t.Fatalf("has_next = %v, want true", pm["has_next"])
	}
}
