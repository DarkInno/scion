package mail

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

// Options configures the SMTP sender.
type Options struct {
	Host        string        // SMTP server host
	Port        int           // SMTP server port (587 for STARTTLS, 465 for implicit TLS)
	Username    string        // SMTP username
	Password    string        // SMTP password (never logged)
	From        string        // Default From address
	FromName    string        // Default From display name
	Timeout     time.Duration // Connection timeout (default 10s)
	QueueSize   int           // Async send queue size (default 100)
	UseSTARTTLS bool          // Use STARTTLS upgrade (port 587)
	UseTLS      bool          // Use implicit TLS (port 465)
}

// Defaults returns default options. Host and Port must be set by the caller.
func Defaults() Options {
	return Options{
		Port:        587,
		Timeout:     10 * time.Second,
		QueueSize:   100,
		UseSTARTTLS: true,
	}
}

// FromEnv loads options from environment variables.
// SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD,
// SMTP_FROM, SMTP_FROM_NAME, SMTP_TIMEOUT, SMTP_QUEUE_SIZE,
// SMTP_USE_STARTTLS, SMTP_USE_TLS.
func FromEnv() Options {
	o := Defaults()
	if v := os.Getenv("SMTP_HOST"); v != "" {
		o.Host = v
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			o.Port = p
		}
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		o.Username = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		o.Password = v
	}
	if v := os.Getenv("SMTP_FROM"); v != "" {
		o.From = v
	}
	if v := os.Getenv("SMTP_FROM_NAME"); v != "" {
		o.FromName = v
	}
	if v := os.Getenv("SMTP_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			o.Timeout = d
		}
	}
	if v := os.Getenv("SMTP_QUEUE_SIZE"); v != "" {
		if s, err := strconv.Atoi(v); err == nil && s > 0 {
			o.QueueSize = s
		}
	}
	if v := os.Getenv("SMTP_USE_STARTTLS"); v != "" {
		o.UseSTARTTLS = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("SMTP_USE_TLS"); v != "" {
		o.UseTLS = strings.ToLower(v) == "true"
	}
	return o
}

// Validate checks that required fields are set.
func (o Options) Validate() error {
	if o.Host == "" {
		return errors.New("SMTP host is required")
	}
	if o.Port <= 0 || o.Port > 65535 {
		return errors.New("SMTP port must be between 1 and 65535")
	}
	if o.From == "" {
		return errors.New("from address is required")
	}
	if o.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if o.QueueSize <= 0 {
		return errors.New("queue size must be positive")
	}
	return nil
}
