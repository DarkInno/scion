package metrics

import (
	"strings"
	"testing"
)

func TestNewRegistersCollectors(t *testing.T) {
	m, err := New(Options{Namespace: "app", Subsystem: "api"})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	m.observe("GET", "/users/{id}", 200, 0.1)
	families, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	found := false
	for _, family := range families {
		if family.GetName() == "app_api_requests_total" {
			found = true
		}
	}
	if !found {
		t.Fatalf("requests metric not found")
	}
}

func TestRegisterDefaultsIsIdempotent(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := m.RegisterDefaults(); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := m.RegisterDefaults(); err != nil {
		t.Fatalf("second register: %v", err)
	}
}

func TestRouteCardinalityOverflow(t *testing.T) {
	m, err := New(Options{MaxRoutes: 1})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if got := m.safeRoute("/first"); got != "/first" {
		t.Fatalf("first route = %q", got)
	}
	if got := m.safeRoute("/second"); got != overflowRoute {
		t.Fatalf("second route = %q", got)
	}
}

func TestSafeMethodRejectsUnexpectedChars(t *testing.T) {
	got := safeMethod("GET\r\n", Defaults())
	if got != "UNKNOWN" {
		t.Fatalf("method = %q", got)
	}
	if strings.ContainsAny(got, "\r\n\x00") {
		t.Fatalf("unsafe method = %q", got)
	}
}
