// Package problem writes RFC 9457-style HTTP API error responses.
package problem

import (
	"os"
	"strconv"
	"strings"
)

// Options controls response sanitization and metadata.
type Options struct {
	// TypeBase is prepended to relative problem types such as "validation".
	TypeBase string
	// MaxDetailLen caps Problem.Detail.
	MaxDetailLen int
	// MaxErrors caps validation error entries.
	MaxErrors int
	// MaxFieldLen caps all other string fields.
	MaxFieldLen int
	// IncludeRequestID copies a safe request ID into the response extensions.
	IncludeRequestID bool
	// RequestIDHeader names the request ID header to read when IncludeRequestID
	// is true.
	RequestIDHeader string
}

// Defaults returns safe defaults for public API errors.
func Defaults() Options {
	return Options{
		MaxDetailLen:    1024,
		MaxErrors:       32,
		MaxFieldLen:     256,
		RequestIDHeader: "X-Request-ID",
	}
}

// FromEnv reads options from environment variables.
//
// Supported variables:
//   - PROBLEM_TYPE_BASE
//   - PROBLEM_MAX_DETAIL_LEN
//   - PROBLEM_MAX_ERRORS
//   - PROBLEM_INCLUDE_REQUEST_ID
//   - PROBLEM_REQUEST_ID_HEADER
func FromEnv() Options {
	o := Defaults()
	if v := os.Getenv("PROBLEM_TYPE_BASE"); v != "" {
		o.TypeBase = v
	}
	if v := os.Getenv("PROBLEM_MAX_DETAIL_LEN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			o.MaxDetailLen = n
		}
	}
	if v := os.Getenv("PROBLEM_MAX_ERRORS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			o.MaxErrors = n
		}
	}
	if v := os.Getenv("PROBLEM_INCLUDE_REQUEST_ID"); v != "" {
		o.IncludeRequestID = strings.EqualFold(v, "true")
	}
	if v := os.Getenv("PROBLEM_REQUEST_ID_HEADER"); v != "" {
		o.RequestIDHeader = v
	}
	return o
}

func (o Options) normalize() Options {
	d := Defaults()
	if o.MaxDetailLen <= 0 {
		o.MaxDetailLen = d.MaxDetailLen
	}
	if o.MaxDetailLen > 8192 {
		o.MaxDetailLen = 8192
	}
	if o.MaxErrors <= 0 {
		o.MaxErrors = d.MaxErrors
	}
	if o.MaxErrors > 256 {
		o.MaxErrors = 256
	}
	if o.MaxFieldLen <= 0 {
		o.MaxFieldLen = d.MaxFieldLen
	}
	if o.MaxFieldLen > 1024 {
		o.MaxFieldLen = 1024
	}
	if o.RequestIDHeader == "" {
		o.RequestIDHeader = d.RequestIDHeader
	}
	return o
}
