package pagination

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponseMarshalNilDataAsArray(t *testing.T) {
	data, err := json.Marshal(PaginatedResult[string]{})
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(data), `"data":[]`) {
		t.Fatalf("nil data should marshal as []: %s", data)
	}
}

func TestResponseWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	PaginatedResult[int]{Data: []int{1}}.WriteJSON(rec)
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), `"data":[1]`) {
		t.Fatalf("body = %q", rec.Body.String())
	}
}
