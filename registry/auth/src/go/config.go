package auth

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	minJWTSecretLength = 32
	maxJWTSecretLength = 512
	maxTokenExpiry     = 7 * 24 * 3600 // 7 days in seconds

	minBcryptCost = 10 // do not go below 10 in production
	maxBcryptCost = 15 // do not go above 15 to avoid DoS via excessive CPU usage
)

// Config holds all auth-related configuration.
// Copy this file into your project and adapt env var names if needed.
type Config struct {
	JWTSecret     string
	TokenExpiry   time.Duration
	DBURL         string
	Issuer        string // JWT issuer claim for token isolation between services
	BcryptCost    int    // bcrypt cost factor, defaults to DefaultBCryptCost in password.go
	OAuthGoogleID string // optional
	OAuthGithubID string // optional
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load() // optional: load .env file if present

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(secret) < minJWTSecretLength {
		return nil, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength)
	}
	if len(secret) > maxJWTSecretLength {
		return nil, fmt.Errorf("JWT_SECRET must not exceed %d characters", maxJWTSecretLength)
	}

	expirySec, _ := strconv.Atoi(os.Getenv("TOKEN_EXPIRY"))
	if expirySec <= 0 {
		expirySec = 3600
	}
	if expirySec > maxTokenExpiry {
		expirySec = maxTokenExpiry
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL is required")
	}

	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "Scion-auth"
	}

	cost, _ := strconv.Atoi(os.Getenv("BCRYPT_COST"))
	if cost < minBcryptCost || cost > maxBcryptCost {
		cost = DefaultBCryptCost
	}

	return &Config{
		JWTSecret:     secret,
		TokenExpiry:   time.Duration(expirySec) * time.Second,
		DBURL:         dbURL,
		Issuer:        issuer,
		BcryptCost:    cost,
		OAuthGoogleID: os.Getenv("OAUTH_GOOGLE_CLIENT_ID"),
		OAuthGithubID: os.Getenv("OAUTH_GITHUB_CLIENT_ID"),
	}, nil
}
