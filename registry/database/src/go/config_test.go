package database

import (
	"strings"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	opts := Defaults()
	want := ApplyPoolStrategy(Options{
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		PingTimeout:     5 * time.Second,
	}, RuntimePoolStrategy(PoolBalanced))
	if opts.MaxOpenConns != want.MaxOpenConns {
		t.Fatalf("MaxOpenConns = %d", opts.MaxOpenConns)
	}
	if opts.MaxIdleConns != want.MaxIdleConns {
		t.Fatalf("MaxIdleConns = %d", opts.MaxIdleConns)
	}
	if opts.ConnMaxLifetime != 30*time.Minute {
		t.Fatalf("ConnMaxLifetime = %s", opts.ConnMaxLifetime)
	}
	if opts.ConnMaxIdleTime != 10*time.Minute {
		t.Fatalf("ConnMaxIdleTime = %s", opts.ConnMaxIdleTime)
	}
	if opts.PingTimeout != 5*time.Second {
		t.Fatalf("PingTimeout = %s", opts.PingTimeout)
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("DATABASE_DRIVER", "scion")
	t.Setenv("DATABASE_DSN", "memory")
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "7")
	t.Setenv("DATABASE_MAX_IDLE_CONNS", "3")
	t.Setenv("DATABASE_CONN_MAX_LIFETIME", "2m")
	t.Setenv("DATABASE_CONN_MAX_IDLE_TIME", "30s")
	t.Setenv("DATABASE_PING_TIMEOUT", "1s")

	opts := FromEnv()
	if opts.DriverName != "scion" || opts.DSN != "memory" {
		t.Fatalf("env driver/dsn not loaded: %#v", opts)
	}
	if opts.MaxOpenConns != 7 || opts.MaxIdleConns != 3 {
		t.Fatalf("env pool not loaded: %#v", opts)
	}
	if opts.ConnMaxLifetime != 2*time.Minute || opts.ConnMaxIdleTime != 30*time.Second || opts.PingTimeout != time.Second {
		t.Fatalf("env durations not loaded: %#v", opts)
	}
}

func TestFromEnvIgnoresInvalidValues(t *testing.T) {
	defaults := Defaults()
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "-1")
	t.Setenv("DATABASE_MAX_IDLE_CONNS", "-1")
	t.Setenv("DATABASE_CONN_MAX_LIFETIME", "bad")
	t.Setenv("DATABASE_CONN_MAX_IDLE_TIME", "-1s")
	t.Setenv("DATABASE_PING_TIMEOUT", "0s")

	opts := FromEnv()
	if opts.MaxOpenConns != defaults.MaxOpenConns || opts.MaxIdleConns != defaults.MaxIdleConns {
		t.Fatalf("invalid ints should retain defaults: %#v", opts)
	}
	if opts.ConnMaxLifetime != 30*time.Minute || opts.ConnMaxIdleTime != 10*time.Minute || opts.PingTimeout != 5*time.Second {
		t.Fatalf("invalid durations should retain defaults: %#v", opts)
	}
}

func TestFromEnvExplicitMaxOpenClampsDerivedIdle(t *testing.T) {
	t.Setenv("DATABASE_POOL_PROFILE", "io-heavy")
	t.Setenv("DATABASE_POOL_CPU_CORES", "4")
	t.Setenv("DATABASE_IO_PARALLELISM", "4")
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "5")

	opts := FromEnv()
	if opts.MaxOpenConns != 5 {
		t.Fatalf("MaxOpenConns = %d", opts.MaxOpenConns)
	}
	if opts.MaxIdleConns != 5 {
		t.Fatalf("MaxIdleConns = %d", opts.MaxIdleConns)
	}
}

func TestOptionsValidate(t *testing.T) {
	opts := Defaults()
	opts.DriverName = "scion"
	opts.DSN = "memory"
	if err := opts.Validate(); err != nil {
		t.Fatalf("valid options: %v", err)
	}

	tests := []struct {
		name string
		opts Options
	}{
		{"missing driver", Options{DSN: "memory", MaxOpenConns: 1, PingTimeout: time.Second}},
		{"missing dsn", Options{DriverName: "scion", MaxOpenConns: 1, PingTimeout: time.Second}},
		{"crlf driver", Options{DriverName: "scion\r\nbad", DSN: "memory", MaxOpenConns: 1, PingTimeout: time.Second}},
		{"null dsn", Options{DriverName: "scion", DSN: "mem\x00ory", MaxOpenConns: 1, PingTimeout: time.Second}},
		{"long driver", Options{DriverName: strings.Repeat("a", maxDriverNameLen+1), DSN: "memory", MaxOpenConns: 1, PingTimeout: time.Second}},
		{"idle exceeds open", Options{DriverName: "scion", DSN: "memory", MaxOpenConns: 1, MaxIdleConns: 2, PingTimeout: time.Second}},
		{"bad ping", Options{DriverName: "scion", DSN: "memory", MaxOpenConns: 1, PingTimeout: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.opts.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeFillsZeroPoolValues(t *testing.T) {
	opts := Options{DriverName: "scion", DSN: "memory"}.normalize()
	defaults := Defaults()
	if opts.MaxOpenConns != defaults.MaxOpenConns || opts.MaxIdleConns != defaults.MaxIdleConns || opts.PingTimeout != 5*time.Second {
		t.Fatalf("normalize did not fill defaults: %#v", opts)
	}
}

func TestNormalizeAllowsZeroIdleWhenMaxOpenSet(t *testing.T) {
	opts := Options{DriverName: "scion", DSN: "memory", MaxOpenConns: 8, MaxIdleConns: 0}.normalize()
	if opts.MaxOpenConns != 8 {
		t.Fatalf("MaxOpenConns = %d", opts.MaxOpenConns)
	}
	if opts.MaxIdleConns != 0 {
		t.Fatalf("MaxIdleConns = %d", opts.MaxIdleConns)
	}
}
