package auth

import (
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// DefaultBCryptCost is the bcrypt cost factor.
// Increase to 13+ for higher security (slower hashing, more CPU).
// Decrease to 10 for faster hashing at the cost of weaker hashes.
const DefaultBCryptCost = 12

// bcryptMaxInput is the maximum number of bytes bcrypt processes.
// Bytes beyond 72 are silently truncated, which can cause two
// different passwords to produce the same hash.
const bcryptMaxInput = 72

// HashPassword hashes a plain text password using bcrypt with the given cost.
// Returns an error if the password exceeds bcrypt's 72-byte input limit.
//
// IMPORTANT: bcrypt silently truncates input at 72 bytes. If you need to
// support longer passwords, pre-hash with SHA-256 before bcrypt:
//
//	h := sha256.Sum256([]byte(password))
//	HashPassword(hex.EncodeToString(h[:]), cost)
func HashPassword(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = DefaultBCryptCost
	}

	if len(password) > bcryptMaxInput {
		return "", fmt.Errorf("password exceeds bcrypt %d-byte input limit; use a pre-hash strategy", bcryptMaxInput)
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}

// CheckPassword compares a plain text password with a hashed password.
// bcrypt.CompareHashAndPassword uses constant-time comparison internally.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SecureCompare performs constant-time comparison of two strings.
// Use this for comparing non-bcrypt secrets (API keys, tokens).
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
