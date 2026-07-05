package crud

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// maxRequestBodySize limits request body to 1MB to prevent DoS.
const maxRequestBodySize = 1 << 20 // 1MB

// EntityStore defines the interface for CRUD operations.
// Adapt this to your actual database layer (GORM, sqlx, pgx, etc.).
//
// IMPORTANT: Implementations must not return (nil, nil) for Create, GetByID,
// or Update. Return a zero-value entity and a sentinel error when no record
// is found (e.g., sql.ErrNoRows, gorm.ErrRecordNotFound).
type EntityStore[T any] interface {
	Create(entity *T) (*T, error)
	GetByID(id uint) (*T, error)
	List(params ListParams) ([]T, int64, error)
	Update(id uint, entity *T) (*T, error)
	Delete(id uint) error
}

type Handler[T any] struct {
	store         EntityStore[T]
	config        *Config
	logger        *slog.Logger
	sortValidator AllowedSortField
	filterAllowed map[string]bool // allowed filter keys; nil = no filtering
}

func NewHandler[T any](store EntityStore[T], cfg *Config) *Handler[T] {
	return &Handler[T]{
		store:         store,
		config:        cfg,
		logger:        slog.Default(),
		sortValidator: DefaultSortValidator,
	}
}

// WithLogger replaces the default logger.
func (h *Handler[T]) WithLogger(logger *slog.Logger) *Handler[T] {
	h.logger = logger
	return h
}

// WithSortValidator sets the function that validates sort field names.
// Required — without this, all sort parameters will be rejected.
// Passing nil resets to DefaultSortValidator.
func (h *Handler[T]) WithSortValidator(fn AllowedSortField) *Handler[T] {
	if fn == nil {
		fn = DefaultSortValidator
	}
	h.sortValidator = fn
	return h
}

// WithFilterFields restricts which query parameters can be used as filters.
// Pass nil to disable filtering entirely.
func (h *Handler[T]) WithFilterFields(allowed map[string]bool) *Handler[T] {
	h.filterAllowed = allowed
	return h
}

// maxFilterValueLen limits the length of individual filter values to prevent
// abuse via excessively long query parameters.
const maxFilterValueLen = 256

// Create handles POST requests.
func (h *Handler[T]) Create(w http.ResponseWriter, r *http.Request) {
	var entity T
	if err := decodeBody(r, &entity); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.store.Create(&entity)
	if err != nil {
		h.logger.Error("failed to create entity", "error", err)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusCreated, created)
}

// Get handles GET requests by ID.
func (h *Handler[T]) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	entity, err := h.store.GetByID(uint(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "entity not found")
		return
	}

	respondJSON(w, http.StatusOK, entity)
}

// List handles GET requests with pagination, filtering, and sorting.
func (h *Handler[T]) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	offset, _ := strconv.Atoi(query.Get("offset"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit > h.config.MaxPageSize {
		limit = h.config.MaxPageSize
	}

	sort := query.Get("sort")
	if sort != "" && !h.sortValidator(sortFieldFromRaw(sort)) {
		respondError(w, http.StatusBadRequest, "invalid sort field")
		return
	}

	// Extract filter params
	filter := make(map[string]string)
	for key, values := range query {
		if key == "offset" || key == "limit" || key == "sort" {
			continue
		}
		if len(values) > 0 {
			if h.filterAllowed != nil && !h.filterAllowed[key] {
				respondError(w, http.StatusBadRequest, "invalid filter field: "+key)
				return
			}
			// Limit filter value length to prevent abuse.
			val := values[0]
			if len(val) > maxFilterValueLen {
				respondError(w, http.StatusBadRequest, "filter value too long")
				return
			}
			filter[key] = val
		}
	}

	params := ParseListParams(offset, limit, h.config.MaxPageSize, sort, filter)

	data, total, err := h.store.List(params)
	if err != nil {
		h.logger.Error("failed to list entities", "error", err)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Ensure data is never null in JSON (use empty slice)
	if data == nil {
		data = make([]T, 0)
	}

	respondJSON(w, http.StatusOK, PaginatedResponse[T]{
		Offset: params.Offset,
		Limit:  params.Limit,
		Total:  total,
		Data:   data,
	})
}

// Update handles PUT requests by ID.
func (h *Handler[T]) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var entity T
	if err := decodeBody(r, &entity); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.store.Update(uint(id), &entity)
	if err != nil {
		h.logger.Error("failed to update entity", "error", err)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE requests by ID.
func (h *Handler[T]) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.store.Delete(uint(id)); err != nil {
		h.logger.Error("failed to delete entity", "error", err)
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// sortFieldFromRaw extracts the field name from a raw sort string (strips "-" prefix).
func sortFieldFromRaw(raw string) string {
	if len(raw) > 0 && raw[0] == '-' {
		return raw[1:]
	}
	return raw
}

// decodeBody reads and size-limits the request body, then decodes JSON.
func decodeBody(r *http.Request, dst interface{}) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.Unmarshal(body, dst)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Default().Error("failed to encode JSON response", "error", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
