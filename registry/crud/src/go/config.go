package crud

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds CRUD-related configuration.
type Config struct {
	DBURL           string
	DefaultPageSize int
	MaxPageSize     int
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL is required")
	}

	maxSize, _ := strconv.Atoi(os.Getenv("MAX_PAGE_SIZE"))
	if maxSize <= 0 {
		maxSize = 100
	}

	defaultSize, _ := strconv.Atoi(os.Getenv("DEFAULT_PAGE_SIZE"))
	if defaultSize <= 0 {
		defaultSize = 20
	}

	// DefaultPageSize must not exceed MaxPageSize
	if defaultSize > maxSize {
		defaultSize = maxSize
	}

	return &Config{
		DBURL:           dbURL,
		DefaultPageSize: defaultSize,
		MaxPageSize:     maxSize,
	}, nil
}

// AllowedSortField is a function that checks if a sort field name is permitted.
// This prevents SQL injection via user-supplied sort parameters.
// Return true if the field is allowed, false otherwise.
//
// Example implementation:
//
//	allowedSort := map[string]bool{"name": true, "created_at": true, "price": true}
//	handler.WithSortValidator(func(field string) bool { return allowedSort[field] })
type AllowedSortField func(field string) bool

// DefaultSortValidator rejects all sort fields. Must be replaced via WithSortValidator.
func DefaultSortValidator(_ string) bool {
	return false
}
