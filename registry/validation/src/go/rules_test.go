package validation

import "testing"

func TestRulesBuiltIns(t *testing.T) {
	rules := []struct {
		name string
		rule Rule
		ok   string
		bad  string
	}{
		{"email", emailRule{}, "a@example.com", "bad"},
		{"url", urlRule{}, "https://example.com", "javascript:alert(1)"},
		{"uuid", uuidRule{}, "550e8400-e29b-41d4-a716-446655440000", "bad"},
		{"ip", ipRule{}, "127.0.0.1", "bad"},
		{"in", newInRule([]string{"red"}), "red", "blue"},
		{"regex", newRegexRule(`^[a-z]+$`, MaxRegexLength), "abc", "123"},
	}
	for _, tc := range rules {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.rule.Validate(tc.ok, true); err != nil {
				t.Fatalf("ok value rejected: %v", err)
			}
			if err := tc.rule.Validate(tc.bad, true); err == nil {
				t.Fatal("bad value accepted")
			}
		})
	}
}
