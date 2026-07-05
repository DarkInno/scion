package crud

import (
	"testing"
)

func TestParseListParams(t *testing.T) {
	tests := []struct {
		name       string
		offset     int
		limit      int
		maxLimit   int
		sort       string
		filter     map[string]string
		wantOffset int
		wantLimit  int
		wantSort   SortField
	}{
		{
			name:   "defaults",
			offset: 0, limit: 0, maxLimit: 100,
			sort: "", filter: nil,
			wantOffset: 0, wantLimit: 20, wantSort: SortField{},
		},
		{
			name:   "custom values",
			offset: 10, limit: 50, maxLimit: 100,
			sort: "-created_at", filter: map[string]string{"status": "active"},
			wantOffset: 10, wantLimit: 50,
			wantSort: SortField{Field: "created_at", Desc: true},
		},
		{
			name:   "limit capped",
			offset: 0, limit: 200, maxLimit: 100,
			sort: "", filter: nil,
			wantOffset: 0, wantLimit: 100, wantSort: SortField{},
		},
		{
			name:   "negative offset clamped",
			offset: -5, limit: 20, maxLimit: 100,
			sort: "", filter: nil,
			wantOffset: 0, wantLimit: 20, wantSort: SortField{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseListParams(tt.offset, tt.limit, tt.maxLimit, tt.sort, tt.filter)
			if got.Offset != tt.wantOffset {
				t.Errorf("Offset = %d, want %d", got.Offset, tt.wantOffset)
			}
			if got.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", got.Limit, tt.wantLimit)
			}
			if got.Sort != tt.wantSort {
				t.Errorf("Sort = %+v, want %+v", got.Sort, tt.wantSort)
			}
		})
	}
}

func TestParseSortField(t *testing.T) {
	tests := []struct {
		input string
		want  SortField
	}{
		{"", SortField{}},
		{"name", SortField{Field: "name", Desc: false}},
		{"-created_at", SortField{Field: "created_at", Desc: true}},
		{"  -price  ", SortField{Field: "price", Desc: true}}, // whitespace trimmed by ParseSortField
		{"-", SortField{Field: "", Desc: true}},
	}

	for _, tt := range tests {
		got := ParseSortField(tt.input)
		if got != tt.want {
			t.Errorf("ParseSortField(%q) = %+v, want %+v", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeFilter(t *testing.T) {
	filter := map[string]string{
		"name":   "foo",
		"status": "active",
		"evil":   "injection",
	}
	allowed := map[string]bool{"name": true, "status": true}

	got := SanitizeFilter(filter, allowed)
	if len(got) != 2 {
		t.Errorf("expected 2 keys, got %d", len(got))
	}
	if got["evil"] != "" {
		t.Error("expected evil key to be removed")
	}
	if got["name"] != "foo" {
		t.Error("expected name key to be preserved")
	}
}

func TestSanitizeFilter_NilAllowed(t *testing.T) {
	filter := map[string]string{"name": "foo"}
	got := SanitizeFilter(filter, nil)
	if got != nil {
		t.Error("expected nil when allowed is nil")
	}
}

func TestFilteredKeys(t *testing.T) {
	filter := map[string]string{
		"z": "1",
		"a": "2",
		"m": "3",
	}
	got := FilteredKeys(filter)
	want := []string{"a", "m", "z"}
	if len(got) != len(want) {
		t.Fatalf("expected %d keys, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("key[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestFilteredKeys_Empty(t *testing.T) {
	got := FilteredKeys(map[string]string{})
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestValidateFilter(t *testing.T) {
	allowed := map[string]bool{"name": true, "status": true}

	if err := ValidateFilter(map[string]string{"name": "foo"}, allowed); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if err := ValidateFilter(map[string]string{"evil": "foo"}, allowed); err == nil {
		t.Error("expected error for disallowed key")
	}
}
