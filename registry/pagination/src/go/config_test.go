package pagination

import "testing"

func TestConfigDefaultsAndNormalize(t *testing.T) {
	defaults := Defaults()
	if defaults.DefaultLimit != 20 || defaults.MaxLimit != 100 || defaults.MaxCursorLen != 256 {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	opts := (Options{DefaultLimit: 500, MaxLimit: 10, MaxOffset: -1}).normalize()
	if opts.DefaultLimit != 10 || opts.MaxLimit != 10 || opts.MaxOffset != 0 || opts.MaxCursorLen != 256 {
		t.Fatalf("normalize failed: %+v", opts)
	}
}
