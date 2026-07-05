package auth

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// User represents the user entity stored in the database.
// Adapt fields to match your database schema.
type User struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"` // use NormalizeEmail() before storing/looking up
	Password  string    `json:"-"`     // never expose in JSON
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RegisterRequest represents the register payload.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest represents the login payload.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the response after successful register or login.
type AuthResponse struct {
	Token string     `json:"token"`
	User  UserPublic `json:"user"`
}

// UserPublic is the safe subset of User fields returned in API responses.
// Password hash is never included.
type UserPublic struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// ToPublic converts a User to UserPublic for API responses.
func (u *User) ToPublic() UserPublic {
	return UserPublic{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}

// errInvalidInput indicates a request failed field-level validation.
var errInvalidInput = errors.New("invalid input")

// simpleEmailRegex checks basic email format.
// Not RFC 5322 compliant, but catches the most common mistakes.
// For production, consider net/mail.ParseAddress or a dedicated validator library.
var simpleEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func validateRegisterRequest(req *RegisterRequest) error {
	req.Email = strings.TrimSpace(req.Email)
	req.Name = strings.TrimSpace(req.Name)
	// SECURITY: Do NOT TrimSpace the password — spaces are valid password
	// characters and contribute to entropy. Trimming would silently weaken
	// passwords that intentionally start or end with spaces.
	if req.Email == "" {
		return fmtWrap(errInvalidInput, "email is required")
	}
	if !simpleEmailRegex.MatchString(req.Email) {
		return fmtWrap(errInvalidInput, "invalid email format")
	}
	if len(req.Email) > 254 {
		return fmtWrap(errInvalidInput, "email too long")
	}
	if utf8.RuneCountInString(req.Password) < 8 {
		return fmtWrap(errInvalidInput, "password must be at least 8 characters")
	}
	if len(req.Password) > bcryptMaxInput {
		return fmtWrap(errInvalidInput, "password too long")
	}
	return nil
}

func validateLoginRequest(req *LoginRequest) error {
	req.Email = strings.TrimSpace(req.Email)
	// SECURITY: Do NOT TrimSpace the password — must match what was registered.
	if req.Email == "" {
		return fmtWrap(errInvalidInput, "email is required")
	}
	if req.Password == "" {
		return fmtWrap(errInvalidInput, "password is required")
	}
	return nil
}

func fmtWrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

// ClientIP extracts the client IP address from a request.
//
// SECURITY: This function does NOT trust X-Forwarded-For or X-Real-Ip headers
// by default, because they are client-controlled and can be spoofed.
// It returns r.RemoteAddr (the TCP source address) directly.
//
// If your service is behind a trusted reverse proxy (nginx, Cloudflare, etc.),
// set TrustedProxyHeaders=true in your Config or use the middleware module's
// TrustedProxy middleware instead.
//
// Relying on spoofable headers for rate limiting allows attackers to bypass
// brute-force protection by sending a different XFF value on each request.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
