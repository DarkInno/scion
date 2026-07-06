package problem

import (
	"net/http"
	"net/url"
	"strings"
)

const mediaType = "application/problem+json"

// InvalidParam describes one invalid request field using a JSON Pointer.
type InvalidParam struct {
	Detail  string `json:"detail"`
	Pointer string `json:"pointer,omitempty"`
}

// Problem is the JSON shape defined by RFC 9457 plus safe extension fields.
type Problem struct {
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Status    int            `json:"status"`
	Detail    string         `json:"detail,omitempty"`
	Instance  string         `json:"instance,omitempty"`
	Errors    []InvalidParam `json:"errors,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

// New creates a problem response. Unsafe fields are sanitized by Write.
func New(status int, title, detail string) Problem {
	return Problem{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	}
}

// Validation creates a 422 validation problem with field-level details.
func Validation(errors []InvalidParam) Problem {
	return Problem{
		Type:   "validation-error",
		Title:  "Request validation failed",
		Status: http.StatusUnprocessableEntity,
		Errors: errors,
	}
}

// Internal creates a generic 500 problem that does not expose implementation
// details.
func Internal() Problem {
	return Problem{
		Type:   "about:blank",
		Title:  http.StatusText(http.StatusInternalServerError),
		Status: http.StatusInternalServerError,
	}
}

func sanitizeProblem(p Problem, opts Options) Problem {
	opts = opts.normalize()
	status := p.Status
	if status < 400 || status > 599 {
		status = http.StatusInternalServerError
	}
	title := safeField(p.Title, opts.MaxFieldLen)
	if title == "" {
		title = http.StatusText(status)
	}
	if title == "" {
		title = "HTTP error"
	}
	return Problem{
		Type:      sanitizeType(p.Type, opts),
		Title:     title,
		Status:    status,
		Detail:    safeField(p.Detail, opts.MaxDetailLen),
		Instance:  sanitizeInstance(p.Instance, opts),
		Errors:    sanitizeErrors(p.Errors, opts),
		RequestID: safeField(p.RequestID, opts.MaxFieldLen),
	}
}

func sanitizeType(value string, opts Options) string {
	if containsUnsafe(value) {
		return "about:blank"
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "about:blank"
	}
	if len(value) > opts.MaxFieldLen {
		return "about:blank"
	}
	if value == "about:blank" {
		return value
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		if _, err := url.ParseRequestURI(value); err == nil {
			return value
		}
		return "about:blank"
	}
	if strings.HasPrefix(value, "/") {
		return value
	}
	if opts.TypeBase != "" && !containsUnsafe(opts.TypeBase) {
		base := strings.TrimRight(opts.TypeBase, "/")
		return base + "/" + strings.TrimLeft(value, "/")
	}
	return "/" + strings.TrimLeft(value, "/")
}

func sanitizeInstance(value string, opts Options) string {
	if containsUnsafe(value) {
		return ""
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) > opts.MaxFieldLen {
		return ""
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return ""
}

func sanitizeErrors(errors []InvalidParam, opts Options) []InvalidParam {
	if len(errors) == 0 {
		return nil
	}
	limit := len(errors)
	if limit > opts.MaxErrors {
		limit = opts.MaxErrors
	}
	out := make([]InvalidParam, 0, limit)
	for _, item := range errors[:limit] {
		detail := safeField(item.Detail, opts.MaxFieldLen)
		if detail == "" {
			continue
		}
		pointer := safePointer(item.Pointer, opts.MaxFieldLen)
		out = append(out, InvalidParam{Detail: detail, Pointer: pointer})
	}
	return out
}

func safeField(value string, maxLen int) string {
	if containsUnsafe(value) {
		return ""
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) > maxLen {
		return value[:maxLen]
	}
	return value
}

func safePointer(value string, maxLen int) string {
	if containsUnsafe(value) {
		return ""
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) > maxLen {
		return ""
	}
	if value == "#" || strings.HasPrefix(value, "#/") || strings.HasPrefix(value, "/") {
		return value
	}
	return ""
}

func containsUnsafe(value string) bool {
	return strings.ContainsAny(value, "\r\n\x00")
}
