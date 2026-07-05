package health

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestChecksValidateConstruction(t *testing.T) {
	if _, err := NewHTTPCheck("local", "http://127.0.0.1"); err == nil || !strings.Contains(err.Error(), "ssrf blocked") {
		t.Fatalf("expected SSRF rejection, got %v", err)
	}
	if _, err := NewTCPCheck("bad\r\nname", "127.0.0.1:80"); err == nil {
		t.Fatal("expected CRLF check name rejection")
	}
	if _, err := NewCustomCheck("nil", nil); err == nil {
		t.Fatal("expected nil custom function rejection")
	}
}

func TestChecksCustomExecute(t *testing.T) {
	check, err := NewCustomCheck("custom", func(ctx context.Context) error {
		return errors.New("down")
	})
	if err != nil {
		t.Fatalf("NewCustomCheck: %v", err)
	}
	result := check.Execute(context.Background())
	if result.Status != StatusFail || result.Error != "down" {
		t.Fatalf("unexpected result: %+v", result)
	}

	okCheck, _ := NewCustomCheck("ok", func(ctx context.Context) error { return nil })
	if got := okCheck.Execute(context.Background()); got.Status != StatusPass {
		t.Fatalf("success result = %+v", got)
	}

	tcp, err := NewTCPCheck("tcp", "127.0.0.1:1", WithTCPTimeout(time.Millisecond))
	if err != nil {
		t.Fatalf("NewTCPCheck: %v", err)
	}
	if tcp.Name() != "tcp" {
		t.Fatalf("tcp name = %q", tcp.Name())
	}
}
