package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// maxRequestBodySize limits request body to 1MB to prevent DoS.
const maxRequestBodySize = 1 << 20 // 1MB

// UserStore defines the interface for user persistence.
// Adapt this to your actual database layer (GORM, sqlx, pgx, etc.)
//
// IMPORTANT: Call NormalizeEmail() on the email parameter before storing or
// looking up users, to ensure case-insensitive matching.
//
// IMPORTANT: Implementations must not return (nil, nil). Return a zero-value
// User and a sentinel error (e.g., sql.ErrNoRows) when no record is found.
type UserStore interface {
	CreateUser(email, passwordHash, name string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id uint) (*User, error)
}

// RateLimiter defines the interface for brute-force protection.
// Implement with in-memory (sync.Map + sliding window) or Redis.
//
// Example: each key gets 10 attempts per 15-minute window.
// On 11th attempt, Allow returns false and the handler returns 429.
type RateLimiter interface {
	Allow(key string) bool
}

type Handler struct {
	store       UserStore
	config      *Config
	rateLimiter RateLimiter
	logger      *slog.Logger
}

func NewHandler(store UserStore, cfg *Config) *Handler {
	return &Handler{
		store:  store,
		config: cfg,
		logger: slog.Default(),
	}
}

// WithRateLimiter adds brute-force protection to the handler.
func (h *Handler) WithRateLimiter(rl RateLimiter) *Handler {
	h.rateLimiter = rl
	return h
}

// WithLogger replaces the default logger.
func (h *Handler) WithLogger(logger *slog.Logger) *Handler {
	h.logger = logger
	return h
}

// decodeBody reads and size-limits the request body, then decodes JSON.
func decodeBody(r *http.Request, dst interface{}) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize+1))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if len(body) > maxRequestBodySize {
		return errors.New("request body too large")
	}
	return json.Unmarshal(body, dst)
}

// Register handles user registration.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateRegisterRequest(&req); err != nil {
		if errors.Is(err, errInvalidInput) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.Email = NormalizeEmail(req.Email)

	hash, err := HashPassword(req.Password, h.config.BcryptCost)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	user, err := h.store.CreateUser(req.Email, hash, req.Name)
	if err != nil {
		// Do not expose whether the error is due to duplicate email or DB failure.
		// Use a generic message to prevent user enumeration.
		h.logger.Warn("create user failed", "email", req.Email, "error", err)
		respondError(w, http.StatusConflict, "unable to create account")
		return
	}
	if user == nil {
		// Defensive: store should never return (nil, nil)
		h.logger.Error("create user returned nil user without error")
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	token, err := GenerateToken(user, h.config.JWTSecret, h.config.TokenExpiry, h.config.Issuer)
	if err != nil {
		h.logger.Error("failed to generate token", "error", err)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusCreated, AuthResponse{Token: token, User: user.ToPublic()})
}

// Login handles user login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeBody(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateLoginRequest(&req); err != nil {
		if errors.Is(err, errInvalidInput) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.Email = NormalizeEmail(req.Email)

	// Brute-force protection: rate limit by IP first, then by email.
	// Order matters: checking IP first prevents incrementing the email
	// counter when the IP is already blocked. This ensures legitimate
	// users sharing an IP (NAT, corporate network) are not penalized
	// by a blocked attacker's email attempts.
	if h.rateLimiter != nil {
		if !h.rateLimiter.Allow("login:ip:" + ClientIP(r)) {
			respondError(w, http.StatusTooManyRequests, "too many attempts, try again later")
			return
		}
		if !h.rateLimiter.Allow("login:" + req.Email) {
			respondError(w, http.StatusTooManyRequests, "too many attempts, try again later")
			return
		}
	}

	user, err := h.store.GetUserByEmail(req.Email)
	if err != nil {
		// Do NOT reveal whether the email exists.
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !CheckPassword(req.Password, user.Password) {
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := GenerateToken(user, h.config.JWTSecret, h.config.TokenExpiry, h.config.Issuer)
	if err != nil {
		h.logger.Error("failed to generate token", "error", err)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, AuthResponse{Token: token, User: user.ToPublic()})
}

// Me returns the current authenticated user.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(claimsKey).(*Claims)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.store.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	if user == nil {
		// Defensive: store should never return (nil, nil)
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user.ToPublic())
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Response headers already sent; cannot return a different status code.
		// Best we can do is record the failure.
		slog.Default().Error("failed to encode JSON response", "error", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
