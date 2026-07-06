package database

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func testColumns() ColumnMap {
	return ColumnMap{
		"id":         "users.id",
		"email":      "users.email",
		"created_at": "users.created_at",
		"status":     "users.status",
	}
}

func TestPlaceholders(t *testing.T) {
	if QuestionPlaceholder(12) != "?" {
		t.Fatal("question placeholder should ignore position")
	}
	if DollarPlaceholder(2) != "$2" {
		t.Fatal("dollar placeholder mismatch")
	}
	if DollarPlaceholder(0) != "$1" {
		t.Fatal("dollar placeholder should clamp to $1")
	}
}

func TestOrderBy(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"", ""},
		{"created_at", "ORDER BY users.created_at ASC"},
		{"-created_at", "ORDER BY users.created_at DESC"},
		{" email ", "ORDER BY users.email ASC"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := OrderBy(tt.raw, testColumns())
			if err != nil {
				t.Fatalf("OrderBy: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestOrderByRejectsUnsafeInput(t *testing.T) {
	tests := []string{
		"created_at DESC",
		"created_at;DROP TABLE users",
		"missing",
		"bad\r\nfield",
		strings.Repeat("a", maxFieldLen+1),
	}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if _, err := OrderBy(raw, testColumns()); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestWhereEqual(t *testing.T) {
	sql, args, err := WhereEqual(map[string]string{
		"status": "active",
		"email":  "ada@example.com",
	}, testColumns(), DollarPlaceholder)
	if err != nil {
		t.Fatalf("WhereEqual: %v", err)
	}
	if sql != "WHERE users.email = $1 AND users.status = $2" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"ada@example.com", "active"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuilder(t *testing.T) {
	b := NewBuilder(testColumns(), QuestionPlaceholder)
	if err := b.WhereEqual("email", "ada@example.com"); err != nil {
		t.Fatalf("WhereEqual email: %v", err)
	}
	if err := b.WhereEqual("status", "active"); err != nil {
		t.Fatalf("WhereEqual status: %v", err)
	}
	if err := b.OrderBy("-created_at"); err != nil {
		t.Fatalf("OrderBy: %v", err)
	}
	sql, args, err := b.SQL()
	if err != nil {
		t.Fatalf("SQL: %v", err)
	}
	if sql != "WHERE users.email = ? AND users.status = ? ORDER BY users.created_at DESC" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"ada@example.com", "active"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuilderDefaultsPlaceholder(t *testing.T) {
	b := NewBuilder(testColumns(), nil)
	if err := b.WhereEqual("id", 123); err != nil {
		t.Fatalf("WhereEqual: %v", err)
	}
	sql, args, err := b.WhereSQL()
	if err != nil {
		t.Fatalf("WhereSQL: %v", err)
	}
	if sql != "WHERE users.id = ?" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{123}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuilderReset(t *testing.T) {
	b := NewBuilder(testColumns(), DollarPlaceholder)
	if err := b.WhereEqual("email", "ada@example.com"); err != nil {
		t.Fatalf("WhereEqual: %v", err)
	}
	if err := b.OrderBy("-created_at"); err != nil {
		t.Fatalf("OrderBy: %v", err)
	}
	b.Reset()
	sql, args, err := b.SQL()
	if err != nil {
		t.Fatalf("SQL after reset: %v", err)
	}
	if sql != "" || args != nil {
		t.Fatalf("reset SQL = %q args=%#v", sql, args)
	}
	if err := b.WhereEqual("status", "active"); err != nil {
		t.Fatalf("WhereEqual after reset: %v", err)
	}
	sql, args, err = b.WhereSQL()
	if err != nil {
		t.Fatalf("WhereSQL after reset: %v", err)
	}
	if sql != "WHERE users.status = $1" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"active"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestWhereEqualRejectsTooManyFilters(t *testing.T) {
	filters := make(map[string]string, maxConditions+1)
	for i := 0; i < maxConditions+1; i++ {
		filters[fmt.Sprintf("f%d", i)] = "x"
	}
	if _, _, err := WhereEqual(filters, testColumns(), QuestionPlaceholder); err == nil {
		t.Fatal("expected too many filters error")
	}
}

func TestBuilderRejectsTooManyConditions(t *testing.T) {
	columns := make(ColumnMap, maxConditions+1)
	for i := 0; i < maxConditions+1; i++ {
		name := fmt.Sprintf("f%d", i)
		columns[name] = "users." + name
	}
	b := NewBuilder(columns, QuestionPlaceholder)
	for i := 0; i < maxConditions; i++ {
		name := fmt.Sprintf("f%d", i)
		if err := b.WhereEqual(name, "x"); err != nil {
			t.Fatalf("WhereEqual %d: %v", i, err)
		}
	}
	if err := b.WhereEqual(fmt.Sprintf("f%d", maxConditions), "x"); err == nil {
		t.Fatal("expected too many conditions error")
	}
}

func TestUnsafeWhitelistColumnRejected(t *testing.T) {
	columns := ColumnMap{"name": "users.name; DROP TABLE users"}
	if _, err := OrderBy("name", columns); err == nil {
		t.Fatal("expected unsafe whitelist error")
	}
}

func TestColumnWhitelistRejectsNonPortableIdentifiers(t *testing.T) {
	tests := []ColumnMap{
		{"name": "users.名字"},
		{"name": "app.public.users.name"},
	}
	for _, columns := range tests {
		if _, err := OrderBy("name", columns); err == nil {
			t.Fatalf("expected unsafe whitelist error for %#v", columns)
		}
	}
}

func TestWhereEqualRejectsUnsafeValue(t *testing.T) {
	if _, _, err := WhereEqual(map[string]string{"email": "a\r\nb"}, testColumns(), QuestionPlaceholder); err == nil {
		t.Fatal("expected CRLF value error")
	}
	if _, _, err := WhereEqual(map[string]string{"email": strings.Repeat("a", maxValueLen+1)}, testColumns(), QuestionPlaceholder); err == nil {
		t.Fatal("expected long value error")
	}
}

func TestBuilderRejectsOversizedByteValue(t *testing.T) {
	b := NewBuilder(testColumns(), QuestionPlaceholder)
	if err := b.WhereEqual("email", bytes.Repeat([]byte("a"), maxValueLen+1)); err == nil {
		t.Fatal("expected long byte value error")
	}
}
