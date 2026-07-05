package middleware

import (
	"os"
	"testing"
	"time"
)

func TestRecoveryDefaults(t *testing.T) {
	d := RecoveryDefaults()
	if d.StackSize != 32 {
		t.Errorf("expected default StackSize 32, got %d", d.StackSize)
	}
}

func TestTimeoutDefaults(t *testing.T) {
	d := TimeoutDefaults()
	if d.Timeout != 30*time.Second {
		t.Errorf("expected default Timeout 30s, got %v", d.Timeout)
	}
}

func TestRequestIDDefaults(t *testing.T) {
	d := RequestIDDefaults()
	if d.HeaderName != "X-Request-ID" {
		t.Errorf("expected default HeaderName X-Request-ID, got %s", d.HeaderName)
	}
}

func TestAccessLogDefaults(t *testing.T) {
	d := AccessLogDefaults()
	if d.BufferSize != 100 {
		t.Errorf("expected default BufferSize 100, got %d", d.BufferSize)
	}
	if !d.DropOnFull {
		t.Error("expected default DropOnFull to be true")
	}
}

func TestCORSDefaults(t *testing.T) {
	d := CORSDefaults()
	if len(d.AllowedMethods) == 0 {
		t.Error("expected default AllowedMethods to be non-empty")
	}
	if len(d.AllowedHeaders) == 0 {
		t.Error("expected default AllowedHeaders to be non-empty")
	}
	if d.MaxAge != 86400 {
		t.Errorf("expected default MaxAge 86400, got %d", d.MaxAge)
	}
}

func TestProxyDefaults(t *testing.T) {
	d := ProxyDefaults()
	if len(d.TrustedProxies) != 0 {
		t.Error("expected default TrustedProxies to be empty")
	}
	if d.ProxyCount != 0 {
		t.Errorf("expected default ProxyCount 0, got %d", d.ProxyCount)
	}
}

func TestBodyLimitDefaults(t *testing.T) {
	d := BodyLimitDefaults()
	if d.MaxSize != 1<<20 {
		t.Errorf("expected default MaxSize 1MB, got %d", d.MaxSize)
	}
}

func TestTraceDefaults(t *testing.T) {
	d := TraceDefaults()
	if d.HeaderName != "traceparent" {
		t.Errorf("expected default HeaderName traceparent, got %s", d.HeaderName)
	}
}

func TestParseEnvDuration(t *testing.T) {
	os.Setenv("TEST_DURATION", "5m")
	defer os.Unsetenv("TEST_DURATION")

	if d := ParseEnvDuration("TEST_DURATION", time.Second); d != 5*time.Minute {
		t.Errorf("expected 5m, got %v", d)
	}
	if d := ParseEnvDuration("MISSING", time.Second); d != time.Second {
		t.Errorf("expected default 1s, got %v", d)
	}
	os.Setenv("TEST_DURATION_BAD", "invalid")
	defer os.Unsetenv("TEST_DURATION_BAD")
	if d := ParseEnvDuration("TEST_DURATION_BAD", time.Second); d != time.Second {
		t.Errorf("expected default for invalid, got %v", d)
	}
}

func TestParseEnvString(t *testing.T) {
	os.Setenv("TEST_STRING", "hello")
	defer os.Unsetenv("TEST_STRING")

	if s := ParseEnvString("TEST_STRING", "default"); s != "hello" {
		t.Errorf("expected hello, got %s", s)
	}
	if s := ParseEnvString("MISSING", "default"); s != "default" {
		t.Errorf("expected default, got %s", s)
	}
}

func TestParseEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	if n := ParseEnvInt("TEST_INT", 0, 0, 100); n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
	if n := ParseEnvInt("MISSING", 10, 0, 100); n != 10 {
		t.Errorf("expected default 10, got %d", n)
	}
	os.Setenv("TEST_INT_BAD", "not-a-number")
	defer os.Unsetenv("TEST_INT_BAD")
	if n := ParseEnvInt("TEST_INT_BAD", 10, 0, 100); n != 10 {
		t.Errorf("expected default for invalid, got %d", n)
	}
	os.Setenv("TEST_INT_LOW", "-5")
	defer os.Unsetenv("TEST_INT_LOW")
	if n := ParseEnvInt("TEST_INT_LOW", 0, 0, 100); n != 0 {
		t.Errorf("expected min 0, got %d", n)
	}
	os.Setenv("TEST_INT_HIGH", "200")
	defer os.Unsetenv("TEST_INT_HIGH")
	if n := ParseEnvInt("TEST_INT_HIGH", 0, 0, 100); n != 100 {
		t.Errorf("expected max 100, got %d", n)
	}
}

func TestParseEnvStringSlice(t *testing.T) {
	os.Setenv("TEST_SLICE", "a, b, c")
	defer os.Unsetenv("TEST_SLICE")

	s := ParseEnvStringSlice("TEST_SLICE", []string{"default"})
	if len(s) != 3 || s[0] != "a" || s[1] != "b" || s[2] != "c" {
		t.Errorf("expected [a b c], got %v", s)
	}

	defaultSlice := []string{"default"}
	if s := ParseEnvStringSlice("MISSING", defaultSlice); len(s) != 1 || s[0] != "default" {
		t.Errorf("expected default slice, got %v", s)
	}

	os.Setenv("TEST_SLICE_EMPTY", "")
	defer os.Unsetenv("TEST_SLICE_EMPTY")
	if s := ParseEnvStringSlice("TEST_SLICE_EMPTY", defaultSlice); len(s) != 1 || s[0] != "default" {
		t.Errorf("expected default for empty env, got %v", s)
	}
}
