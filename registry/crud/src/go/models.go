package crud

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BaseEntity provides common fields for all entities.
// Embed this into your entity models.
// ORM-specific struct tags (gorm, db, etc.) should be added by the user
// when they adapt the code to their project.
type BaseEntity struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PaginatedResponse represents a paginated list response.
// Data is guaranteed to be a non-nil slice (empty, not null).
type PaginatedResponse[T any] struct {
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
	Data   []T   `json:"data"`
}

// ListParams represents query parameters for list endpoints.
type ListParams struct {
	Offset int
	Limit  int
	Sort   SortField
	Filter map[string]string
}

// SortField represents a parsed sort directive.
type SortField struct {
	Field string // column name
	Desc  bool   // true = descending (prefix "-")
}

// ParseListParams extracts and validates pagination, sort, and filter from query values.
//
// sortFormat: raw sort string, e.g. "-created_at" or "name"
//
// This function does NOT validate sort field names against your schema.
// Use Handler.sortValidator (set via WithSortValidator) to restrict allowed fields.
func ParseListParams(offset, limit, maxLimit int, sort string, filter map[string]string) ListParams {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	return ListParams{
		Offset: offset,
		Limit:  limit,
		Sort:   ParseSortField(sort),
		Filter: filter,
	}
}

// ParseSortField parses a sort string like "-created_at" into a SortField.
// Returns a zero-value SortField if the input is empty.
func ParseSortField(raw string) SortField {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return SortField{}
	}

	if raw[0] == '-' {
		return SortField{Field: raw[1:], Desc: true}
	}
	return SortField{Field: raw, Desc: false}
}

// SanitizeFilter removes any filter keys not in the allowed set.
// Pass nil to disable filtering entirely.
func SanitizeFilter(filter map[string]string, allowed map[string]bool) map[string]string {
	if allowed == nil {
		return nil
	}
	result := make(map[string]string, len(filter))
	for key, value := range filter {
		if allowed[key] {
			result[key] = value
		}
	}
	return result
}

// FilteredKeys returns sorted filter keys for deterministic SQL generation.
func FilteredKeys(filter map[string]string) []string {
	keys := make([]string, 0, len(filter))
	for k := range filter {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ValidateFilter checks that all filter keys are in the allowed set.
// Returns an error listing the first disallowed key found.
func ValidateFilter(filter map[string]string, allowed map[string]bool) error {
	for key := range filter {
		if !allowed[key] {
			return fmt.Errorf("filter field not allowed: %s", key)
		}
	}
	return nil
}
