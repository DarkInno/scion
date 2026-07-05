package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"normal password", "mySecurePassword123", false},
		{"empty password", "", false},
		{"unicode password", "密码123!@#", false},
		{"exactly 72 bytes", strings.Repeat("a", 72), false},
		{"73 bytes too long", strings.Repeat("a", 73), true},
		{"long unicode within limit", strings.Repeat("密", 24), false}, // 3 bytes each = 72
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password, DefaultBCryptCost)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash == "" {
				t.Error("expected non-empty hash")
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	password := "mySecurePassword123"
	hash, err := HashPassword(password, DefaultBCryptCost)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}
	if CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
	if CheckPassword("", hash) {
		t.Error("CheckPassword should return false for empty password")
	}
}

func TestHashPassword_CostBounds(t *testing.T) {
	password := "testpassword"

	// Too low cost should be corrected
	hashLow, err := HashPassword(password, 5)
	if err != nil {
		t.Fatalf("HashPassword with low cost failed: %v", err)
	}
	if !CheckPassword(password, hashLow) {
		t.Error("password check failed for low-cost hash")
	}

	// Too high cost should be corrected
	hashHigh, err := HashPassword(password, 20)
	if err != nil {
		t.Fatalf("HashPassword with high cost failed: %v", err)
	}
	if !CheckPassword(password, hashHigh) {
		t.Error("password check failed for high-cost hash")
	}
}

func TestHashPassword_ProducesDifferentHashes(t *testing.T) {
	password := "samepassword"
	hash1, err := HashPassword(password, DefaultBCryptCost)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	hash2, err := HashPassword(password, DefaultBCryptCost)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash1 == hash2 {
		t.Error("bcrypt should produce different hashes for the same password due to salt")
	}
}

func TestSecureCompare(t *testing.T) {
	if !SecureCompare("secret", "secret") {
		t.Error("SecureCompare should return true for equal strings")
	}
	if SecureCompare("secret", "different") {
		t.Error("SecureCompare should return false for different strings")
	}
	if !SecureCompare("", "") {
		t.Error("SecureCompare should return true for two empty strings")
	}
}

func TestBcryptMaxInputConstant(t *testing.T) {
	if bcryptMaxInput != 72 {
		t.Errorf("bcryptMaxInput should be 72, got %d", bcryptMaxInput)
	}
	// Verify bcrypt library agrees
	if bcrypt.MaxCost < 15 {
		t.Error("bcrypt.MaxCost unexpectedly low")
	}
}
