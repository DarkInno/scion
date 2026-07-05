package validation

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorsJSONAndWriteJSON(t *testing.T) {
	ve := NewValidationError()
	if ve.HasErrors() {
		t.Fatal("new error should be empty")
	}
	ve.Add("email", "is invalid")
	if !ve.HasErrors() || !strings.Contains(ve.Error(), "email") {
		t.Fatalf("unexpected error state: %v", ve)
	}
	data, err := json.Marshal(ve)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(data), `"email"`) {
		t.Fatalf("json = %s", data)
	}

	rec := httptest.NewRecorder()
	if err := ve.WriteJSON(rec, 422); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if rec.Code != 422 || !strings.Contains(rec.Body.String(), "email") {
		t.Fatalf("response code=%d body=%q", rec.Code, rec.Body.String())
	}
}
