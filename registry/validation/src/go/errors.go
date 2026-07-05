package validation

import (
	"encoding/json"
	"strings"
)

// ValidationError is a structured, field-keyed collection of validation error
// messages. It implements the error interface and can be serialized directly
// to JSON so that HTTP handlers can return it to clients unchanged.
//
// The canonical JSON shape is:
//
//	{"errors":{"field":["message one","message two"]}}
type ValidationError struct {
	Errors map[string][]string `json:"errors"`
}

// NewValidationError returns an empty ValidationError ready to collect errors.
func NewValidationError() *ValidationError {
	return &ValidationError{Errors: make(map[string][]string)}
}

// Add appends a message to the list of errors for the given field.
func (ve *ValidationError) Add(field, message string) {
	if ve.Errors == nil {
		ve.Errors = make(map[string][]string)
	}
	ve.Errors[field] = append(ve.Errors[field], message)
}

// HasErrors reports whether the ValidationError contains at least one error.
func (ve *ValidationError) HasErrors() bool {
	return ve != nil && len(ve.Errors) > 0
}

// Error implements the error interface, producing a human-readable summary of
// all field errors.
func (ve *ValidationError) Error() string {
	if !ve.HasErrors() {
		return "validation passed"
	}
	var b strings.Builder
	b.WriteString("validation failed")
	// Stable-ish ordering is not required for an error string, but iterating
	// the map is sufficient for a diagnostic message.
	for field, msgs := range ve.Errors {
		b.WriteString("; ")
		b.WriteString(field)
		b.WriteString(": ")
		b.WriteString(strings.Join(msgs, ", "))
	}
	return b.String()
}

// MarshalJSON implements json.Marshaler. It guarantees that the errors map is
// never serialized as null (an empty object is emitted instead) so that
// clients always receive a stable, predictable shape.
func (ve ValidationError) MarshalJSON() ([]byte, error) {
	type alias ValidationError
	tmp := ve
	if tmp.Errors == nil {
		tmp.Errors = map[string][]string{}
	}
	return json.Marshal((alias)(tmp))
}

// UnmarshalJSON implements json.Unmarshaler for symmetry with MarshalJSON. It
// ensures the errors map is never left as nil after decoding.
func (ve *ValidationError) UnmarshalJSON(data []byte) error {
	type alias ValidationError
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	ve.Errors = a.Errors
	if ve.Errors == nil {
		ve.Errors = map[string][]string{}
	}
	return nil
}
