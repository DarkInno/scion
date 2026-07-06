package problem

import "testing"

func TestOptionsNormalize(t *testing.T) {
	opts := (Options{}).normalize()
	if opts.MaxDetailLen != 1024 || opts.MaxErrors != 32 || opts.RequestIDHeader != "X-Request-ID" {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("PROBLEM_TYPE_BASE", "https://api.example.com/problems")
	t.Setenv("PROBLEM_MAX_DETAIL_LEN", "128")
	t.Setenv("PROBLEM_MAX_ERRORS", "4")
	t.Setenv("PROBLEM_INCLUDE_REQUEST_ID", "true")
	t.Setenv("PROBLEM_REQUEST_ID_HEADER", "X-Trace-ID")

	opts := FromEnv().normalize()
	if opts.TypeBase != "https://api.example.com/problems" || opts.MaxDetailLen != 128 {
		t.Fatalf("unexpected env options: %+v", opts)
	}
	if opts.MaxErrors != 4 || !opts.IncludeRequestID || opts.RequestIDHeader != "X-Trace-ID" {
		t.Fatalf("unexpected env options: %+v", opts)
	}
}
