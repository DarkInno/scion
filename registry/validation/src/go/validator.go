package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Security-oriented defaults. These are exported so callers can reference them
// when reasoning about configured limits.
const (
	// MaxValueLength is the default maximum length (in bytes) of a single field
	// value. Values larger than this are rejected to prevent memory-exhaustion
	// attacks.
	MaxValueLength = 1 << 20 // 1 MB

	// MaxRegexLength is the default maximum length (in characters) of a regular
	// expression pattern. Combined with Go's RE2 engine (which matches in
	// linear time and cannot suffer catastrophic backtracking), this keeps
	// regex validation safe from denial-of-service.
	MaxRegexLength = 256
)

// Source identifies where field values are read from in an HTTP request.
type Source int

const (
	// JSONSource reads values from the JSON request body (the default).
	JSONSource Source = iota
	// QuerySource reads values from the URL query string.
	QuerySource
	// FormSource reads values from the parsed form body.
	FormSource
)

// Options configures Builder behavior.
type Options struct {
	// Source selects the request location to validate.
	Source Source
	// MaxValueLength bounds the size of a single field value.
	MaxValueLength int
	// MaxRegexLength bounds the size of a regex pattern.
	MaxRegexLength int
}

// Option configures a Builder via functional options.
type Option func(*Options)

// WithSource sets the request source to validate.
func WithSource(s Source) Option {
	return func(o *Options) { o.Source = s }
}

// WithMaxValueLength overrides the per-value length limit.
func WithMaxValueLength(n int) Option {
	return func(o *Options) { o.MaxValueLength = n }
}

// WithMaxRegexLength overrides the regex pattern length limit.
func WithMaxRegexLength(n int) Option {
	return func(o *Options) { o.MaxRegexLength = n }
}

func applyDefaults(o *Options) {
	if o.MaxValueLength <= 0 {
		o.MaxValueLength = MaxValueLength
	}
	if o.MaxRegexLength <= 0 {
		o.MaxRegexLength = MaxRegexLength
	}
}

// Rule is the interface implemented by every validation rule.
type Rule interface {
	// Validate checks value. present reports whether the field was supplied in
	// the request (as opposed to being absent). Returning a non-nil error
	// records a validation failure for the field.
	Validate(value string, present bool) error
	// Name returns a human-readable identifier for the rule.
	Name() string
}

// requiredMarker is implemented by rules that mark a field as required. It is
// used to short-circuit optional empty fields and to avoid running format
// rules on missing values.
type requiredMarker interface {
	isRequired()
}

// Builder collects fields and their rules and validates requests against them.
// A Builder is safe for concurrent read-only use after it has been fully
// constructed (i.e. no more Field/Rule calls are made).
type Builder struct {
	opts   Options
	fields []*Field
}

// New creates a new Builder with the given options applied over secure
// defaults.
func New(opts ...Option) *Builder {
	o := Options{Source: JSONSource}
	for _, opt := range opts {
		opt(&o)
	}
	applyDefaults(&o)
	return &Builder{opts: o}
}

// withOptions returns a shallow copy of the Builder whose options have been
// overridden. The fields slice is shared (read-only) so this is cheap and safe.
func (b *Builder) withOptions(opts []Option) *Builder {
	cp := *b
	for _, opt := range opts {
		opt(&cp.opts)
	}
	applyDefaults(&cp.opts)
	return &cp
}

// Field begins a new field declaration and returns a *Field for chaining.
func (b *Builder) Field(name string) *Field {
	f := &Field{name: name, builder: b}
	b.fields = append(b.fields, f)
	return f
}

// Field is a single field declaration together with its associated rules.
type Field struct {
	name    string
	rules   []Rule
	builder *Builder
}

// Field begins a new field declaration, allowing chaining straight from the
// previous field, e.g. validation.New().Field("a").Required().Field("b").Min(1).
func (f *Field) Field(name string) *Field {
	return f.builder.Field(name)
}

// Required adds a rule that the field must be present and non-empty.
func (f *Field) Required() *Field {
	f.rules = append(f.rules, requiredRule{})
	return f
}

// Min adds a rule that the numeric value must be greater than or equal to n.
func (f *Field) Min(n int) *Field {
	f.rules = append(f.rules, minRule{n: float64(n)})
	return f
}

// Max adds a rule that the numeric value must be less than or equal to n.
func (f *Field) Max(n int) *Field {
	f.rules = append(f.rules, maxRule{n: float64(n)})
	return f
}

// Length adds a rule that the string length must be within [min, max].
func (f *Field) Length(min, max int) *Field {
	f.rules = append(f.rules, lengthRule{min: min, max: max})
	return f
}

// Email adds a rule that the value is a valid email address.
func (f *Field) Email() *Field {
	f.rules = append(f.rules, emailRule{})
	return f
}

// URL adds a rule that the value is a valid http(s) URL.
func (f *Field) URL() *Field {
	f.rules = append(f.rules, urlRule{})
	return f
}

// UUID adds a rule that the value is a valid (hyphenated) UUID.
func (f *Field) UUID() *Field {
	f.rules = append(f.rules, uuidRule{})
	return f
}

// IP adds a rule that the value is a valid IPv4 or IPv6 address.
func (f *Field) IP() *Field {
	f.rules = append(f.rules, ipRule{})
	return f
}

// In adds a rule that the value must be one of the given values.
func (f *Field) In(values ...string) *Field {
	f.rules = append(f.rules, newInRule(values))
	return f
}

// Regex adds a rule that the value must match the given regular expression.
// The pattern is compiled once when the rule is created; its length is limited
// by Options.MaxRegexLength.
func (f *Field) Regex(pattern string) *Field {
	f.rules = append(f.rules, newRegexRule(pattern, f.builder.opts.MaxRegexLength))
	return f
}

// Custom adds a rule that delegates validation to the provided function. The
// function is only invoked when the field is present and non-empty.
func (f *Field) Custom(name string, fn func(string) error) *Field {
	f.rules = append(f.rules, customRule{name: name, fn: fn})
	return f
}

// hasRequired reports whether the field has a Required rule attached.
func (f *Field) hasRequired() bool {
	for _, r := range f.rules {
		if _, ok := r.(requiredMarker); ok {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Validation entry points are defined on *Builder. The following methods are
// delegated from *Field so that the terminal field of a fluent chain can also
// drive validation, e.g. validation.New().Field("x").Required().ValidateJSON(r).
// ---------------------------------------------------------------------------

// ValidateValues delegates to the underlying Builder.
func (f *Field) ValidateValues(getter func(string) (string, bool)) *ValidationError {
	return f.builder.ValidateValues(getter)
}

// ValidateRequest delegates to the underlying Builder.
func (f *Field) ValidateRequest(r *http.Request) *ValidationError {
	return f.builder.ValidateRequest(r)
}

// ValidateJSON delegates to the underlying Builder.
func (f *Field) ValidateJSON(r *http.Request) *ValidationError {
	return f.builder.ValidateJSON(r)
}

// ValidateQuery delegates to the underlying Builder.
func (f *Field) ValidateQuery(r *http.Request) *ValidationError {
	return f.builder.ValidateQuery(r)
}

// ValidateForm delegates to the underlying Builder.
func (f *Field) ValidateForm(r *http.Request) *ValidationError {
	return f.builder.ValidateForm(r)
}

// ---------------------------------------------------------------------------
// Core rules (Required / Min / Max / Length). The remaining built-in rules
// live in rules.go.
// ---------------------------------------------------------------------------

type requiredRule struct{}

func (requiredRule) isRequired() {}
func (requiredRule) Validate(value string, present bool) error {
	if !present || strings.TrimSpace(value) == "" {
		return fmt.Errorf("is required")
	}
	return nil
}
func (requiredRule) Name() string { return "required" }

type minRule struct{ n float64 }

func (r minRule) Validate(value string, present bool) error {
	if !present || value == "" {
		return nil
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	if v < r.n {
		return fmt.Errorf("must be at least %s", formatNumber(r.n))
	}
	return nil
}
func (minRule) Name() string { return "min" }

type maxRule struct{ n float64 }

func (r maxRule) Validate(value string, present bool) error {
	if !present || value == "" {
		return nil
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	if v > r.n {
		return fmt.Errorf("must be at most %s", formatNumber(r.n))
	}
	return nil
}
func (maxRule) Name() string { return "max" }

type lengthRule struct{ min, max int }

func (r lengthRule) Validate(value string, present bool) error {
	if !present || value == "" {
		return nil
	}
	l := len(value)
	if l < r.min || l > r.max {
		return fmt.Errorf("length must be between %d and %d characters", r.min, r.max)
	}
	return nil
}
func (lengthRule) Name() string { return "length" }

// formatNumber renders a numeric bound without a trailing decimal point.
func formatNumber(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// ---------------------------------------------------------------------------
// Validation entry points.
// ---------------------------------------------------------------------------

// ValidateValues validates a value source accessed through getter. getter
// returns the field value and whether it was present. This is the core
// validation routine used by ValidateJSON/ValidateQuery/ValidateForm.
func (b *Builder) ValidateValues(getter func(string) (string, bool)) *ValidationError {
	ve := NewValidationError()
	for _, f := range b.fields {
		value, present := getter(f.name)

		// Enforce global safety constraints before running any rule.
		if err := checkValueSafety(value, b.opts.MaxValueLength); err != nil {
			ve.Add(f.name, err.Error())
			continue
		}

		// "Effectively empty" means absent or whitespace-only. Required uses
		// trimmed emptiness so that a whitespace-only value still fails.
		trimmedEmpty := !present || strings.TrimSpace(value) == ""
		hasRequired := f.hasRequired()

		// Optional and effectively empty: nothing to validate.
		if trimmedEmpty && !hasRequired {
			continue
		}

		// Effectively empty but required: only the Required rule can
		// meaningfully run; running format rules on whitespace would produce
		// noise.
		if trimmedEmpty {
			for _, rule := range f.rules {
				if _, ok := rule.(requiredMarker); !ok {
					continue
				}
				if err := rule.Validate(value, present); err != nil {
					ve.Add(f.name, err.Error())
				}
			}
			continue
		}

		// Non-empty value: run every rule. Required will pass; format rules
		// perform their checks.
		for _, rule := range f.rules {
			if err := rule.Validate(value, present); err != nil {
				ve.Add(f.name, err.Error())
			}
		}
	}
	return ve
}

// ValidateRequest validates the request using the Builder's configured Source.
func (b *Builder) ValidateRequest(r *http.Request) *ValidationError {
	switch b.opts.Source {
	case QuerySource:
		return b.ValidateQuery(r)
	case FormSource:
		return b.ValidateForm(r)
	default:
		return b.ValidateJSON(r)
	}
}

// ValidateQuery validates values from the URL query string.
func (b *Builder) ValidateQuery(r *http.Request) *ValidationError {
	q := r.URL.Query()
	return b.ValidateValues(func(field string) (string, bool) {
		vs := q[field]
		if len(vs) == 0 {
			return "", false
		}
		return vs[0], true
	})
}

// ValidateForm validates values from the parsed form body.
func (b *Builder) ValidateForm(r *http.Request) *ValidationError {
	if err := r.ParseForm(); err != nil {
		ve := NewValidationError()
		ve.Add("_form", fmt.Sprintf("could not parse form: %v", err))
		return ve
	}
	form := r.PostForm
	return b.ValidateValues(func(field string) (string, bool) {
		vs := form[field]
		if len(vs) == 0 {
			return "", false
		}
		return vs[0], true
	})
}

// ValidateJSON validates values from the JSON request body. The body is read
// and then reset (r.Body is replaced with a fresh reader over the same bytes)
// so that downstream handlers may re-read it.
func (b *Builder) ValidateJSON(r *http.Request) *ValidationError {
	if r.Body == nil {
		r.Body = http.NoBody
	}
	max := int64(b.opts.MaxValueLength)
	body, err := io.ReadAll(io.LimitReader(r.Body, max+1))
	_ = r.Body.Close()
	// Reset the body so downstream handlers can re-read it.
	r.Body = io.NopCloser(bytes.NewReader(body))

	if err != nil {
		ve := NewValidationError()
		ve.Add("_body", fmt.Sprintf("could not read request body: %v", err))
		return ve
	}
	if int64(len(body)) > max {
		ve := NewValidationError()
		ve.Add("_body", fmt.Sprintf("request body exceeds maximum length of %d bytes", max))
		return ve
	}

	values, err := decodeJSONValues(body)
	if err != nil {
		ve := NewValidationError()
		ve.Add("_body", fmt.Sprintf("invalid JSON: %v", err))
		return ve
	}

	return b.ValidateValues(func(field string) (string, bool) {
		v, ok := values[field]
		return v, ok
	})
}

// decodeJSONValues decodes a JSON object body into a flat map of field name to
// string value. Non-string scalars are converted to their string form; null is
// treated as absent; objects and arrays are kept as their raw JSON text.
func decodeJSONValues(body []byte) (map[string]string, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]string{}, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	values := make(map[string]string, len(raw))
	for k, v := range raw {
		s, ok, err := rawMessageToString(v)
		if err != nil {
			return nil, fmt.Errorf("field %q: %v", k, err)
		}
		if !ok {
			// null => absent
			continue
		}
		values[k] = s
	}
	return values, nil
}

// rawMessageToString converts a single JSON value (as a RawMessage) into a
// string. The boolean reports whether the value should be considered present
// (false for null/absent).
func rawMessageToString(v json.RawMessage) (string, bool, error) {
	trim := bytes.TrimSpace(v)
	if len(trim) == 0 {
		return "", false, nil
	}
	if bytes.Equal(trim, []byte("null")) {
		return "", false, nil
	}
	switch trim[0] {
	case '"':
		var s string
		if err := json.Unmarshal(trim, &s); err != nil {
			return "", false, err
		}
		return s, true, nil
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		// json.Number preserves the original textual representation, which is
		// what Min/Max expect to parse back.
		var n json.Number
		if err := json.Unmarshal(trim, &n); err != nil {
			return "", false, err
		}
		return string(n), true, nil
	default:
		// bool, object, or array: keep the raw JSON text.
		return string(trim), true, nil
	}
}

// checkValueSafety enforces global safety constraints on a field value:
// length limit, no null bytes, no CR/LF characters. These guards run before
// any rule and protect against memory-exhaustion, header injection, and log
// injection attacks.
func checkValueSafety(value string, maxLen int) error {
	if len(value) > maxLen {
		return fmt.Errorf("value exceeds maximum length of %d bytes", maxLen)
	}
	if strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("value must not contain null bytes")
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("value must not contain CR or LF characters")
	}
	return nil
}
