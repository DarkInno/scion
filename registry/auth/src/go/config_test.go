package auth

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_Valid(t *testing.T) {
	os.Setenv("JWT_SECRET", "this_is_a_very_long_secret_key_that_is_at_least_32")
	os.Setenv("DB_URL", "postgres://localhost/test")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DB_URL")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.JWTSecret == "" {
		t.Error("expected JWTSecret to be set")
	}
	if cfg.TokenExpiry != time.Hour {
		t.Errorf("expected default expiry 1h, got %v", cfg.TokenExpiry)
	}
	if cfg.Issuer != "Scion-auth" {
		t.Errorf("expected default issuer, got %s", cfg.Issuer)
	}
	if cfg.BcryptCost != DefaultBCryptCost {
		t.Errorf("expected default bcrypt cost %d, got %d", DefaultBCryptCost, cfg.BcryptCost)
	}
}

func TestLoadConfig_MissingJWTSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	os.Setenv("DB_URL", "postgres://localhost/test")
	defer os.Unsetenv("DB_URL")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing JWT_SECRET")
	}
}

func TestLoadConfig_ShortSecret(t *testing.T) {
	os.Setenv("JWT_SECRET", "short")
	os.Setenv("DB_URL", "postgres://localhost/test")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DB_URL")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for short JWT_SECRET")
	}
}

func TestLoadConfig_LongSecret(t *testing.T) {
	os.Setenv("JWT_SECRET", "this_is_a_very_long_secret_key_that_is_at_least_32")
	os.Setenv("DB_URL", "postgres://localhost/test")
	// Create a secret longer than maxJWTSecretLength (512)
	longSecret := make([]byte, maxJWTSecretLength+1)
	for i := range longSecret {
		longSecret[i] = 'a'
	}
	os.Setenv("JWT_SECRET", string(longSecret))
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DB_URL")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for overly long JWT_SECRET")
	}
}

func TestLoadConfig_MissingDBURL(t *testing.T) {
	os.Setenv("JWT_SECRET", "this_is_a_very_long_secret_key_that_is_at_least_32")
	os.Unsetenv("DB_URL")
	defer os.Unsetenv("JWT_SECRET")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing DB_URL")
	}
}

func TestLoadConfig_CustomExpiry(t *testing.T) {
	os.Setenv("JWT_SECRET", "this_is_a_very_long_secret_key_that_is_at_least_32")
	os.Setenv("DB_URL", "postgres://localhost/test")
	os.Setenv("TOKEN_EXPIRY", "7200")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DB_URL")
	defer os.Unsetenv("TOKEN_EXPIRY")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.TokenExpiry != 2*time.Hour {
		t.Errorf("expected expiry 2h, got %v", cfg.TokenExpiry)
	}
}

func TestLoadConfig_ExpiryCapped(t *testing.T) {
	os.Setenv("JWT_SECRET", "this_is_a_very_long_secret_key_that_is_at_least_32")
	os.Setenv("DB_URL", "postgres://localhost/test")
	os.Setenv("TOKEN_EXPIRY", "9999999")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DB_URL")
	defer os.Unsetenv("TOKEN_EXPIRY")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	maxDur := time.Duration(maxTokenExpiry) * time.Second
	if cfg.TokenExpiry != maxDur {
		t.Errorf("expected expiry capped at %v, got %v", maxDur, cfg.TokenExpiry)
	}
}

func TestLoadConfig_CustomIssuer(t *testing.T) {
	os.Setenv("JWT_SECRET", "this_is_a_very_long_secret_key_that_is_at_least_32")
	os.Setenv("DB_URL", "postgres://localhost/test")
	os.Setenv("JWT_ISSUER", "my-service")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DB_URL")
	defer os.Unsetenv("JWT_ISSUER")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Issuer != "my-service" {
		t.Errorf("expected issuer 'my-service', got %s", cfg.Issuer)
	}
}

func TestLoadConfig_BcryptCostBounds(t *testing.T) {
	os.Setenv("JWT_SECRET", "this_is_a_very_long_secret_key_that_is_at_least_32")
	os.Setenv("DB_URL", "postgres://localhost/test")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DB_URL")

	tests := []struct {
		name     string
		cost     string
		expected int
	}{
		{"too low", "5", DefaultBCryptCost},
		{"too high", "20", DefaultBCryptCost},
		{"valid high", "14", 14},
		{"valid low", "10", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cost != "" {
				os.Setenv("BCRYPT_COST", tt.cost)
				defer os.Unsetenv("BCRYPT_COST")
			} else {
				os.Unsetenv("BCRYPT_COST")
			}

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if cfg.BcryptCost != tt.expected {
				t.Errorf("expected cost %d, got %d", tt.expected, cfg.BcryptCost)
			}
		})
	}
}
