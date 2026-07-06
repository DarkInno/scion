package database

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOpenSuccess(t *testing.T) {
	ensureTestDriver()
	opts := Defaults()
	opts.DriverName = testDriverName
	opts.DSN = "memory"
	opts.MaxOpenConns = 4
	opts.MaxIdleConns = 2

	db, err := Open(context.Background(), opts)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if got := db.Stats().MaxOpenConnections; got != 4 {
		t.Fatalf("MaxOpenConnections = %d", got)
	}
}

func TestOpenAppliesDefaultPoolValues(t *testing.T) {
	ensureTestDriver()
	db, err := Open(context.Background(), Options{
		DriverName: testDriverName,
		DSN:        "memory",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	want := Defaults().MaxOpenConns
	if got := db.Stats().MaxOpenConnections; got != want {
		t.Fatalf("MaxOpenConnections = %d, want %d", got, want)
	}
}

func TestOpenValidationError(t *testing.T) {
	ensureTestDriver()
	_, err := Open(context.Background(), Options{DriverName: testDriverName, DSN: "bad\r\nvalue"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestOpenRejectsNilContext(t *testing.T) {
	if _, err := Open(nil, Options{DriverName: testDriverName, DSN: "memory"}); err == nil {
		t.Fatal("expected nil context error")
	}
}

func TestOpenPingErrorDoesNotLeakDSN(t *testing.T) {
	ensureTestDriver()
	dsn := "ping-error user=app password=supersecret"
	_, err := Open(context.Background(), Options{
		DriverName:  testDriverName,
		DSN:         dsn,
		PingTimeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected ping error")
	}
	if strings.Contains(err.Error(), "supersecret") || strings.Contains(err.Error(), dsn) {
		t.Fatalf("error leaked DSN: %v", err)
	}
}

func TestOpenPingTimeout(t *testing.T) {
	ensureTestDriver()
	_, err := Open(context.Background(), Options{
		DriverName:  testDriverName,
		DSN:         "slow",
		PingTimeout: time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected ping timeout")
	}
	if strings.Contains(err.Error(), "slow") {
		t.Fatalf("error leaked DSN: %v", err)
	}
}
