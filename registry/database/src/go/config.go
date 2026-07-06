// Package database provides small database/sql helpers for copy-paste Go
// backends: safe connection-pool setup, transaction wrapping, and whitelisted
// query fragment construction.
//
// The package uses only the Go standard library. It does not import any SQL
// driver; callers must register the driver they use in their application.
package database

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	maxDriverNameLen = 64
	maxDSNLen        = 4096
)

// Options configures database/sql opening and pool behaviour.
type Options struct {
	// DriverName is the database/sql driver name, such as "postgres", "mysql",
	// or "sqlite3". The driver must be imported by the calling application.
	DriverName string
	// DSN is the driver-specific data source name. It is never included in
	// errors returned by this module.
	DSN string
	// MaxOpenConns limits concurrently open connections. Defaults are derived
	// dynamically from RuntimePoolStrategy(PoolBalanced).
	MaxOpenConns int
	// MaxIdleConns limits idle connections retained by database/sql. Defaults
	// are derived dynamically from RuntimePoolStrategy(PoolBalanced).
	MaxIdleConns int
	// ConnMaxLifetime is the maximum lifetime of one connection. Defaults to 30m.
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum idle lifetime of one connection. Defaults to 10m.
	ConnMaxIdleTime time.Duration
	// PingTimeout bounds the startup ping performed by Open. Defaults to 5s.
	PingTimeout time.Duration
}

// Defaults returns safe dynamic pool defaults. DriverName and DSN must still be
// supplied by the caller or loaded with FromEnv.
func Defaults() Options {
	opts := Options{
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		PingTimeout:     5 * time.Second,
	}
	return ApplyPoolStrategy(opts, RuntimePoolStrategy(PoolBalanced))
}

// FromEnv returns Defaults() overridden by environment variables:
//
//	DATABASE_DRIVER
//	DATABASE_DSN
//	DATABASE_MAX_OPEN_CONNS
//	DATABASE_MAX_IDLE_CONNS
//	DATABASE_CONN_MAX_LIFETIME
//	DATABASE_CONN_MAX_IDLE_TIME
//	DATABASE_PING_TIMEOUT
//	DATABASE_POOL_PROFILE          (conservative, balanced, io-heavy)
//	DATABASE_POOL_CPU_CORES
//	DATABASE_IO_PARALLELISM
//	DATABASE_POOL_MAX_OPEN_LIMIT
//
// Invalid numeric or duration values are ignored and the default is retained.
func FromEnv() Options {
	opts := Defaults()
	maxOpenSet := false
	maxIdleSet := false
	explicitMaxOpen := 0
	explicitMaxIdle := 0

	if v := os.Getenv("DATABASE_DRIVER"); v != "" {
		opts.DriverName = v
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		opts.DSN = v
	}
	if v := os.Getenv("DATABASE_MAX_OPEN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opts.MaxOpenConns = n
			explicitMaxOpen = n
			maxOpenSet = true
		}
	}
	if v := os.Getenv("DATABASE_MAX_IDLE_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			opts.MaxIdleConns = n
			explicitMaxIdle = n
			maxIdleSet = true
		}
	}
	if v := os.Getenv("DATABASE_CONN_MAX_LIFETIME"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= 0 {
			opts.ConnMaxLifetime = d
		}
	}
	if v := os.Getenv("DATABASE_CONN_MAX_IDLE_TIME"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= 0 {
			opts.ConnMaxIdleTime = d
		}
	}
	if v := os.Getenv("DATABASE_PING_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			opts.PingTimeout = d
		}
	}

	strategy := RuntimePoolStrategy(PoolBalanced)
	strategySet := false
	if profile := os.Getenv("DATABASE_POOL_PROFILE"); profile != "" {
		strategy.Profile = PoolProfile(profile)
		strategySet = true
	}
	if v := os.Getenv("DATABASE_POOL_CPU_CORES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			strategy.CPUCores = n
			strategySet = true
		}
	}
	if v := os.Getenv("DATABASE_IO_PARALLELISM"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			strategy.IOParallelism = n
			strategySet = true
		}
	}
	if v := os.Getenv("DATABASE_POOL_MAX_OPEN_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			strategy.MaxOpenLimit = n
			strategySet = true
		}
	}
	if strategySet {
		opts = ApplyPoolStrategy(opts, strategy)
	}
	opts = applyExplicitPoolOverrides(opts, maxOpenSet, explicitMaxOpen, maxIdleSet, explicitMaxIdle)
	return opts
}

// Validate enforces required fields, length limits, and CRLF / null-byte
// rejection for all string options.
func (o Options) Validate() error {
	if err := validateRequiredString("driver name", o.DriverName, maxDriverNameLen); err != nil {
		return err
	}
	if err := validateRequiredString("dsn", o.DSN, maxDSNLen); err != nil {
		return err
	}
	if o.MaxOpenConns <= 0 {
		return errors.New("database: max open connections must be positive")
	}
	if o.MaxIdleConns < 0 {
		return errors.New("database: max idle connections cannot be negative")
	}
	if o.MaxIdleConns > o.MaxOpenConns {
		return errors.New("database: max idle connections cannot exceed max open connections")
	}
	if o.ConnMaxLifetime < 0 {
		return errors.New("database: connection max lifetime cannot be negative")
	}
	if o.ConnMaxIdleTime < 0 {
		return errors.New("database: connection max idle time cannot be negative")
	}
	if o.PingTimeout <= 0 {
		return errors.New("database: ping timeout must be positive")
	}
	return nil
}

func (o Options) normalize() Options {
	defaults := Defaults()
	if o.MaxOpenConns == 0 {
		o.MaxOpenConns = defaults.MaxOpenConns
		if o.MaxIdleConns == 0 {
			o.MaxIdleConns = defaults.MaxIdleConns
		}
	}
	if o.ConnMaxLifetime == 0 {
		o.ConnMaxLifetime = defaults.ConnMaxLifetime
	}
	if o.ConnMaxIdleTime == 0 {
		o.ConnMaxIdleTime = defaults.ConnMaxIdleTime
	}
	if o.PingTimeout == 0 {
		o.PingTimeout = defaults.PingTimeout
	}
	return o
}

func applyExplicitPoolOverrides(opts Options, maxOpenSet bool, maxOpen int, maxIdleSet bool, maxIdle int) Options {
	if maxOpenSet {
		opts.MaxOpenConns = maxOpen
	}
	if maxIdleSet {
		opts.MaxIdleConns = maxIdle
	} else if maxOpenSet && opts.MaxIdleConns > opts.MaxOpenConns {
		opts.MaxIdleConns = opts.MaxOpenConns
	}
	return opts
}

func validateRequiredString(name, value string, maxLen int) error {
	if value == "" {
		return errors.New("database: " + name + " is required")
	}
	return validateString(name, value, maxLen)
}

func validateString(name, value string, maxLen int) error {
	if len(value) > maxLen {
		return errors.New("database: " + name + " is too long")
	}
	if strings.ContainsAny(value, "\r\n\x00") {
		return errors.New("database: " + name + " contains invalid characters")
	}
	return nil
}
