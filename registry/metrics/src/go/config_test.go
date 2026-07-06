package metrics

import "testing"

func TestOptionsNormalize(t *testing.T) {
	opts := (Options{Namespace: "bad-name", Subsystem: "http"}).normalize()
	if opts.Namespace != "scion" || opts.Subsystem != "http" {
		t.Fatalf("unexpected metric parts: %+v", opts)
	}
	if opts.MaxRouteLen != 128 || opts.MaxRoutes != 128 || len(opts.Buckets) == 0 {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("METRICS_NAMESPACE", "app")
	t.Setenv("METRICS_SUBSYSTEM", "api")
	t.Setenv("METRICS_MAX_ROUTE_LEN", "64")
	t.Setenv("METRICS_MAX_METHOD_LEN", "8")
	t.Setenv("METRICS_MAX_ROUTES", "4")

	opts := FromEnv().normalize()
	if opts.Namespace != "app" || opts.Subsystem != "api" {
		t.Fatalf("unexpected names: %+v", opts)
	}
	if opts.MaxRouteLen != 64 || opts.MaxMethodLen != 8 || opts.MaxRoutes != 4 {
		t.Fatalf("unexpected limits: %+v", opts)
	}
}
