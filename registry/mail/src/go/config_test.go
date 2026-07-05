package mail

import (
	"testing"
	"time"
)

func TestConfigDefaultsFromEnvAndValidate(t *testing.T) {
	opts := Defaults()
	if opts.Port != 587 || opts.Timeout != 10*time.Second || !opts.UseSTARTTLS {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
	if err := opts.Validate(); err == nil {
		t.Fatal("missing host/from should fail")
	}

	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_FROM", "noreply@example.com")
	t.Setenv("SMTP_TIMEOUT", "2s")
	t.Setenv("SMTP_QUEUE_SIZE", "3")
	t.Setenv("SMTP_USE_STARTTLS", "false")
	t.Setenv("SMTP_USE_TLS", "true")

	opts = FromEnv()
	if err := opts.Validate(); err != nil {
		t.Fatalf("env options should validate: %v", err)
	}
	if opts.Port != 2525 || opts.Timeout != 2*time.Second || opts.QueueSize != 3 {
		t.Fatalf("env numeric options not applied: %+v", opts)
	}
	if opts.UseSTARTTLS || !opts.UseTLS {
		t.Fatalf("TLS options not applied: %+v", opts)
	}
}
