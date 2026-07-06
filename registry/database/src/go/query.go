package database

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	maxFieldLen   = 128
	maxValueLen   = 1024
	maxConditions = 32
)

// ColumnMap maps user-facing filter/sort keys to trusted SQL column names.
// Column names must be simple identifiers or dot-qualified identifiers such as
// "users.created_at".
type ColumnMap map[string]string

// Placeholder returns a placeholder for the 1-based argument position.
type Placeholder func(position int) string

// QuestionPlaceholder returns "?" for MySQL, SQLite, and other drivers that
// use anonymous placeholders.
func QuestionPlaceholder(_ int) string {
	return "?"
}

// DollarPlaceholder returns PostgreSQL-style placeholders: $1, $2, ...
func DollarPlaceholder(position int) string {
	if position < 1 {
		position = 1
	}
	return "$" + strconv.Itoa(position)
}

// OrderBy converts a client sort key such as "created_at" or "-created_at"
// into a safe ORDER BY fragment using a whitelist. Empty input returns an empty
// fragment and nil error.
func OrderBy(raw string, columns ColumnMap) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if err := validateString("sort field", raw, maxFieldLen); err != nil {
		return "", err
	}

	desc := false
	key := raw
	if strings.HasPrefix(key, "-") {
		desc = true
		key = strings.TrimPrefix(key, "-")
	}
	column, err := lookupColumn(key, columns)
	if err != nil {
		return "", err
	}
	direction := "ASC"
	if desc {
		direction = "DESC"
	}
	return "ORDER BY " + column + " " + direction, nil
}

// WhereEqual builds a deterministic WHERE fragment joined by AND. Filter keys
// are mapped through the whitelist; filter values are returned as args and are
// never concatenated into SQL.
func WhereEqual(filters map[string]string, columns ColumnMap, placeholder Placeholder) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}
	if len(filters) > maxConditions {
		return "", nil, errors.New("database: too many filter conditions")
	}
	b := NewBuilder(columns, placeholder)
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := b.WhereEqual(key, filters[key]); err != nil {
			return "", nil, err
		}
	}
	return b.WhereSQL()
}

// Builder incrementally constructs safe WHERE and ORDER BY fragments.
type Builder struct {
	columns     ColumnMap
	placeholder Placeholder
	where       []string
	args        []any
	order       string
}

// NewBuilder creates a query fragment builder. A nil placeholder defaults to
// QuestionPlaceholder.
func NewBuilder(columns ColumnMap, placeholder Placeholder) *Builder {
	if placeholder == nil {
		placeholder = QuestionPlaceholder
	}
	return &Builder{
		columns:     columns,
		placeholder: placeholder,
	}
}

// WhereEqual adds "column = placeholder" for a whitelisted field.
func (b *Builder) WhereEqual(field string, value any) error {
	if len(b.where) >= maxConditions {
		return errors.New("database: too many filter conditions")
	}
	column, err := lookupColumn(field, b.columns)
	if err != nil {
		return err
	}
	if err := validateArgValue(value); err != nil {
		return err
	}
	b.args = append(b.args, value)
	b.where = append(b.where, column+" = "+b.placeholder(len(b.args)))
	return nil
}

// OrderBy sets the ORDER BY fragment from a client sort key.
func (b *Builder) OrderBy(raw string) error {
	order, err := OrderBy(raw, b.columns)
	if err != nil {
		return err
	}
	b.order = order
	return nil
}

// WhereSQL returns the WHERE fragment and a copy of its args.
func (b *Builder) WhereSQL() (string, []any, error) {
	if len(b.where) == 0 {
		return "", nil, nil
	}
	args := append([]any(nil), b.args...)
	return "WHERE " + strings.Join(b.where, " AND "), args, nil
}

// OrderSQL returns the ORDER BY fragment or an empty string.
func (b *Builder) OrderSQL() string {
	return b.order
}

// Reset clears all accumulated query fragments and arguments. It also drops
// references to old argument values so a reused Builder does not keep request
// data alive longer than necessary.
func (b *Builder) Reset() {
	b.where = nil
	b.args = nil
	b.order = ""
}

// SQL returns the combined WHERE and ORDER BY fragments plus WHERE args.
func (b *Builder) SQL() (string, []any, error) {
	where, args, err := b.WhereSQL()
	if err != nil {
		return "", nil, err
	}
	switch {
	case where == "":
		return b.order, args, nil
	case b.order == "":
		return where, args, nil
	default:
		return where + " " + b.order, args, nil
	}
}

func lookupColumn(field string, columns ColumnMap) (string, error) {
	field = strings.TrimSpace(field)
	if err := validateRequiredString("field", field, maxFieldLen); err != nil {
		return "", err
	}
	if columns == nil {
		return "", errors.New("database: column whitelist is required")
	}
	column, ok := columns[field]
	if !ok {
		return "", fmt.Errorf("database: field is not allowed: %s", field)
	}
	if err := validateColumn(column); err != nil {
		return "", err
	}
	return column, nil
}

func validateColumn(column string) error {
	if err := validateRequiredString("column", column, maxFieldLen); err != nil {
		return err
	}
	parts := strings.Split(column, ".")
	if len(parts) > 3 {
		return errors.New("database: column whitelist contains too many identifier segments")
	}
	for _, part := range parts {
		if !isIdentifier(part) {
			return errors.New("database: column whitelist contains unsafe identifier")
		}
	}
	return nil
}

func validateArgValue(value any) error {
	switch v := value.(type) {
	case string:
		return validateString("filter value", v, maxValueLen)
	case []byte:
		if len(v) > maxValueLen {
			return errors.New("database: filter value is too long")
		}
	}
	return nil
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '_':
			continue
		case i == 0 && isASCIILetter(c):
			continue
		case i > 0 && (isASCIILetter(c) || isASCIIDigit(c)):
			continue
		default:
			return false
		}
	}
	return true
}

func isASCIILetter(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
