package validation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// getterFrom builds a ValueGetter from a flat map. A key present with an empty
// string counts as present; a missing key counts as absent.
func getterFrom(m map[string]string) func(string) (string, bool) {
	return func(field string) (string, bool) {
		v, ok := m[field]
		return v, ok
	}
}

func fieldErrors(ve *ValidationError, field string) []string {
	if ve == nil {
		return nil
	}
	return ve.Errors[field]
}

func hasErrorContaining(ve *ValidationError, field, substr string) bool {
	for _, m := range fieldErrors(ve, field) {
		if strings.Contains(m, substr) {
			return true
		}
	}
	return false
}

func TestChainExample(t *testing.T) {
	// The exact chain from the task description.
	b := New().Field("email").Required().Email().Field("age").Min(18).Max(120)

	ve := b.ValidateValues(getterFrom(map[string]string{
		"email": "user@example.com",
		"age":   "25",
	}))
	if ve.HasErrors() {
		t.Fatalf("expected no errors, got %v", ve)
	}
}

func TestRequired(t *testing.T) {
	b := New().Field("name").Required()

	cases := []struct {
		name   string
		values map[string]string
		want   bool
	}{
		{"present", map[string]string{"name": "Alice"}, false},
		{"empty string", map[string]string{"name": ""}, true},
		{"absent", map[string]string{}, true},
		{"whitespace only", map[string]string{"name": "   "}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ve := b.ValidateValues(getterFrom(c.values))
			if ve.HasErrors() != c.want {
				t.Fatalf("HasErrors=%v want %v (%v)", ve.HasErrors(), c.want, ve)
			}
		})
	}
}

func TestOptionalEmptySkipped(t *testing.T) {
	// An optional field that is absent must not trigger format rules.
	b := New().Field("email").Email()
	ve := b.ValidateValues(getterFrom(map[string]string{}))
	if ve.HasErrors() {
		t.Fatalf("expected no errors for absent optional field, got %v", ve)
	}
	// Present empty optional also skipped.
	ve = b.ValidateValues(getterFrom(map[string]string{"email": ""}))
	if ve.HasErrors() {
		t.Fatalf("expected no errors for empty optional field, got %v", ve)
	}
}

func TestMin(t *testing.T) {
	b := New().Field("age").Min(18)

	cases := []struct {
		value string
		want  bool
	}{
		{"25", false},
		{"18", false},
		{"17", true},
		{"0", true},
		{"-5", true},
		{"18.5", false},
		{"abc", true},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			ve := b.ValidateValues(getterFrom(map[string]string{"age": c.value}))
			if ve.HasErrors() != c.want {
				t.Fatalf("value=%q HasErrors=%v want %v (%v)", c.value, ve.HasErrors(), c.want, ve)
			}
		})
	}
}

func TestMax(t *testing.T) {
	b := New().Field("age").Max(120)
	cases := []struct {
		value string
		want  bool
	}{
		{"25", false},
		{"120", false},
		{"121", true},
		{"200", true},
		{"abc", true},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			ve := b.ValidateValues(getterFrom(map[string]string{"age": c.value}))
			if ve.HasErrors() != c.want {
				t.Fatalf("value=%q HasErrors=%v want %v (%v)", c.value, ve.HasErrors(), c.want, ve)
			}
		})
	}
}

func TestLength(t *testing.T) {
	b := New().Field("code").Length(3, 6)
	cases := []struct {
		value string
		want  bool
	}{
		{"abcd", false},
		{"abc", false},
		{"abcdef", false},
		{"ab", true},
		{"abcdefg", true},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			ve := b.ValidateValues(getterFrom(map[string]string{"code": c.value}))
			if ve.HasErrors() != c.want {
				t.Fatalf("value=%q HasErrors=%v want %v (%v)", c.value, ve.HasErrors(), c.want, ve)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	b := New().Field("email").Email()
	valid := []string{
		"user@example.com",
		"a.b+tag@sub.example.co",
		"User.Name@Example.COM",
	}
	for _, v := range valid {
		t.Run("valid/"+v, func(t *testing.T) {
			ve := b.ValidateValues(getterFrom(map[string]string{"email": v}))
			if ve.HasErrors() {
				t.Fatalf("expected %q to be valid, got %v", v, ve)
			}
		})
	}
	invalid := []string{
		"notanemail",
		"user@@example.com",
		"@example.com",
		"John <john@example.com>",
		"user@",
	}
	for _, v := range invalid {
		t.Run("invalid/"+v, func(t *testing.T) {
			ve := b.ValidateValues(getterFrom(map[string]string{"email": v}))
			if !ve.HasErrors() {
				t.Fatalf("expected %q to be invalid", v)
			}
		})
	}
}

func TestURL(t *testing.T) {
	b := New().Field("site").URL()
	valid := []string{
		"http://example.com",
		"https://example.com/path?q=1",
		"https://example.com:8080",
	}
	for _, v := range valid {
		t.Run("valid/"+v, func(t *testing.T) {
			ve := b.ValidateValues(getterFrom(map[string]string{"site": v}))
			if ve.HasErrors() {
				t.Fatalf("expected %q valid, got %v", v, ve)
			}
		})
	}
	invalid := []string{
		"ftp://example.com",
		"javascript:alert(1)",
		"not a url",
		"example.com",
	}
	for _, v := range invalid {
		t.Run("invalid/"+v, func(t *testing.T) {
			ve := b.ValidateValues(getterFrom(map[string]string{"site": v}))
			if !ve.HasErrors() {
				t.Fatalf("expected %q invalid", v)
			}
		})
	}
}

func TestUUID(t *testing.T) {
	b := New().Field("id").UUID()
	valid := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"00000000-0000-0000-0000-000000000000",
	}
	for _, v := range valid {
		ve := b.ValidateValues(getterFrom(map[string]string{"id": v}))
		if ve.HasErrors() {
			t.Fatalf("expected %q valid, got %v", v, ve)
		}
	}
	invalid := []string{
		"not-a-uuid",
		"550e8400-e29b-41d4-a716-44665544000",
		"GGGe8400-e29b-41d4-a716-446655440000",
	}
	for _, v := range invalid {
		ve := b.ValidateValues(getterFrom(map[string]string{"id": v}))
		if !ve.HasErrors() {
			t.Fatalf("expected %q invalid", v)
		}
	}
}

func TestIP(t *testing.T) {
	b := New().Field("ip").IP()
	valid := []string{"192.168.1.1", "::1", "2001:db8::1", "10.0.0.1"}
	for _, v := range valid {
		ve := b.ValidateValues(getterFrom(map[string]string{"ip": v}))
		if ve.HasErrors() {
			t.Fatalf("expected %q valid, got %v", v, ve)
		}
	}
	invalid := []string{"999.999.999.999", "not an ip", "256.1.1.1"}
	for _, v := range invalid {
		ve := b.ValidateValues(getterFrom(map[string]string{"ip": v}))
		if !ve.HasErrors() {
			t.Fatalf("expected %q invalid", v)
		}
	}
}

func TestIn(t *testing.T) {
	b := New().Field("role").In("admin", "user", "guest")
	if ve := b.ValidateValues(getterFrom(map[string]string{"role": "admin"})); ve.HasErrors() {
		t.Fatalf("unexpected: %v", ve)
	}
	ve := b.ValidateValues(getterFrom(map[string]string{"role": "superadmin"}))
	if !ve.HasErrors() {
		t.Fatal("expected error for superadmin")
	}
	if !hasErrorContaining(ve, "role", "must be one of") {
		t.Fatalf("expected 'must be one of' message, got %v", ve)
	}
}

func TestRegex(t *testing.T) {
	b := New().Field("phone").Regex(`^\d{3}-\d{4}$`)
	if ve := b.ValidateValues(getterFrom(map[string]string{"phone": "123-4567"})); ve.HasErrors() {
		t.Fatalf("unexpected: %v", ve)
	}
	ve := b.ValidateValues(getterFrom(map[string]string{"phone": "12-34"}))
	if !ve.HasErrors() {
		t.Fatal("expected error for 12-34")
	}
}

func TestCustom(t *testing.T) {
	b := New().Field("token").Custom("even", func(v string) error {
		if len(v)%2 != 0 {
			return errExample("must be even length")
		}
		return nil
	})
	if ve := b.ValidateValues(getterFrom(map[string]string{"token": "abcd"})); ve.HasErrors() {
		t.Fatalf("unexpected: %v", ve)
	}
	ve := b.ValidateValues(getterFrom(map[string]string{"token": "abc"}))
	if !hasErrorContaining(ve, "token", "even") {
		t.Fatalf("expected even-length error, got %v", ve)
	}
}

// errExample is a tiny error type to avoid pulling fmt into every test.
type errExample string

func (e errExample) Error() string { return string(e) }

func TestValidateJSON(t *testing.T) {
	b := New().Field("email").Required().Email().Field("age").Min(18).Max(120)

	t.Run("valid", func(t *testing.T) {
		body := `{"email":"user@example.com","age":25}`
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		ve := b.ValidateJSON(r)
		if ve.HasErrors() {
			t.Fatalf("unexpected: %v", ve)
		}
		// Body must be reset so downstream handlers can re-read it.
		rest, _ := io.ReadAll(r.Body)
		if string(rest) != body {
			t.Fatalf("body not reset, got %q", string(rest))
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"email":"notanemail","age":25}`))
		ve := b.ValidateJSON(r)
		if !hasErrorContaining(ve, "email", "email") {
			t.Fatalf("expected email error, got %v", ve)
		}
	})

	t.Run("missing required", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"age":25}`))
		ve := b.ValidateJSON(r)
		if !hasErrorContaining(ve, "email", "required") {
			t.Fatalf("expected required error, got %v", ve)
		}
	})

	t.Run("number as JSON number", func(t *testing.T) {
		// age provided as a JSON number, not a string; Min/Max must still work.
		bn := New().Field("age").Min(18).Max(120)
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"age":25}`))
		if ve := bn.ValidateJSON(r); ve.HasErrors() {
			t.Fatalf("unexpected: %v", ve)
		}
		r2 := httptest.NewRequest("POST", "/", strings.NewReader(`{"age":5}`))
		if ve := bn.ValidateJSON(r2); !hasErrorContaining(ve, "age", "at least") {
			t.Fatalf("expected min error, got %v", ve)
		}
	})

	t.Run("null treated as absent", func(t *testing.T) {
		bn := New().Field("email").Email().Field("age").Min(18)
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"email":null,"age":25}`))
		if ve := bn.ValidateJSON(r); ve.HasErrors() {
			t.Fatalf("null should be treated as absent optional, got %v", ve)
		}
	})

	t.Run("invalid json body", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{bad json`))
		ve := b.ValidateJSON(r)
		if !hasErrorContaining(ve, "_body", "invalid JSON") {
			t.Fatalf("expected invalid JSON error, got %v", ve)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		ve := b.ValidateJSON(r)
		if !hasErrorContaining(ve, "email", "required") {
			t.Fatalf("expected required error on empty body, got %v", ve)
		}
	})
}

func TestValidateQuery(t *testing.T) {
	b := New(WithSource(QuerySource)).Field("q").Required().Field("page").Min(1).Max(100)

	t.Run("valid", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/?q=hello&page=2", nil)
		if ve := b.ValidateRequest(r); ve.HasErrors() {
			t.Fatalf("unexpected: %v", ve)
		}
	})

	t.Run("missing required", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/?page=2", nil)
		ve := b.ValidateRequest(r)
		if !hasErrorContaining(ve, "q", "required") {
			t.Fatalf("expected required error, got %v", ve)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/?q=hi&page=999", nil)
		ve := b.ValidateRequest(r)
		if !hasErrorContaining(ve, "page", "at most") {
			t.Fatalf("expected max error, got %v", ve)
		}
	})
}

func TestValidateForm(t *testing.T) {
	b := New(WithSource(FormSource)).Field("name").Required().Field("age").Min(18)

	t.Run("valid", func(t *testing.T) {
		form := url.Values{}
		form.Set("name", "Alice")
		form.Set("age", "30")
		r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if ve := b.ValidateRequest(r); ve.HasErrors() {
			t.Fatalf("unexpected: %v", ve)
		}
	})

	t.Run("missing required", func(t *testing.T) {
		form := url.Values{}
		form.Set("age", "30")
		r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		ve := b.ValidateRequest(r)
		if !hasErrorContaining(ve, "name", "required") {
			t.Fatalf("expected required error, got %v", ve)
		}
	})
}

func TestMiddleware(t *testing.T) {
	schema := New().Field("email").Required().Email()

	t.Run("valid passes through", func(t *testing.T) {
		called := false
		h := Middleware(schema)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"email":"a@b.com"}`))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if !called {
			t.Fatal("next handler not called")
		}
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid returns 422", func(t *testing.T) {
		called := false
		h := Middleware(schema)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{"email":"bad"}`))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if called {
			t.Fatal("next handler should not be called")
		}
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("expected json content type, got %q", ct)
		}
		var resp ValidationError
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("could not decode response: %v", err)
		}
		if !hasErrorContaining(&resp, "email", "email") {
			t.Fatalf("expected email error in body, got %s", w.Body.String())
		}
	})

	t.Run("nil schema passes through", func(t *testing.T) {
		called := false
		h := Middleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		r := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if !called {
			t.Fatal("nil schema should pass through")
		}
	})

	t.Run("middleware option overrides source", func(t *testing.T) {
		// schema defaults to JSON, but the middleware validates query instead.
		schema := New().Field("q").Required()
		called := false
		h := Middleware(schema, WithSource(QuerySource))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		r := httptest.NewRequest("GET", "/?q=hello", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if !called {
			t.Fatal("expected pass-through for valid query")
		}
	})
}

func TestValidationErrorJSON(t *testing.T) {
	t.Run("with errors", func(t *testing.T) {
		ve := NewValidationError()
		ve.Add("email", "is required")
		ve.Add("email", "must be a valid email address")
		ve.Add("age", "must be at least 18")
		data, err := json.Marshal(ve)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got map[string]map[string][]string
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(got["errors"]["email"]) != 2 {
			t.Fatalf("expected 2 email errors, got %v", got)
		}
		if len(got["errors"]["age"]) != 1 {
			t.Fatalf("expected 1 age error, got %v", got)
		}
	})

	t.Run("empty serializes to empty object", func(t *testing.T) {
		ve := NewValidationError()
		data, err := json.Marshal(ve)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(data) != `{"errors":{}}` {
			t.Fatalf("expected {\"errors\":{}}, got %s", string(data))
		}
	})

	t.Run("nil errors map serializes to empty object", func(t *testing.T) {
		ve := &ValidationError{}
		data, err := json.Marshal(ve)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(data) != `{"errors":{}}` {
			t.Fatalf("expected {\"errors\":{}}, got %s", string(data))
		}
	})

	t.Run("round trip", func(t *testing.T) {
		ve := NewValidationError()
		ve.Add("x", "msg")
		data, _ := json.Marshal(ve)
		var decoded ValidationError
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !decoded.HasErrors() || len(decoded.Errors["x"]) != 1 {
			t.Fatalf("round trip failed: %+v", decoded)
		}
	})

	t.Run("error string", func(t *testing.T) {
		ve := NewValidationError()
		if !strings.Contains(ve.Error(), "validation passed") {
			t.Fatalf("empty error string: %q", ve.Error())
		}
		ve.Add("x", "bad")
		if !strings.Contains(ve.Error(), "validation failed") {
			t.Fatalf("error string: %q", ve.Error())
		}
	})
}

func TestOptionsOverrideMaxValueLength(t *testing.T) {
	b := New(WithMaxValueLength(10)).Field("data").Required()
	// 11 bytes exceeds the configured limit of 10.
	ve := b.ValidateValues(getterFrom(map[string]string{"data": "12345678901"}))
	if !hasErrorContaining(ve, "data", "maximum length") {
		t.Fatalf("expected max length error, got %v", ve)
	}
	// exactly 10 is fine.
	ve = b.ValidateValues(getterFrom(map[string]string{"data": "1234567890"}))
	if ve.HasErrors() {
		t.Fatalf("unexpected: %v", ve)
	}
}

func TestMultipleErrorsCollected(t *testing.T) {
	b := New().Field("email").Required().Email()
	// present but invalid email: only the email format rule runs (required is
	// skipped because the value is non-empty).
	ve := b.ValidateValues(getterFrom(map[string]string{"email": "notanemail"}))
	if !hasErrorContaining(ve, "email", "email") {
		t.Fatalf("expected email format error, got %v", ve)
	}
}
