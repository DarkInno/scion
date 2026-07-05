package crud

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateRoutePrefix_CRUD(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid", "/api/v1/products", "/api/v1/products", false},
		{"trailing slash", "/api/v1/products/", "/api/v1/products", false},
		{"no leading slash", "api/v1/products", "/api/v1/products", false},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateRoutePrefix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRoutePrefix(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateRoutePrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHandler_RegisterRoutes(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	mux := http.NewServeMux()
	err := h.RegisterRoutes(mux, "/api/v1/products")
	if err != nil {
		t.Fatalf("RegisterRoutes failed: %v", err)
	}

	// Test Create
	p := Product{Name: "Widget", Price: 9.99}
	body, _ := json.Marshal(p)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("create route failed: expected %d, got %d", http.StatusCreated, rr.Code)
	}

	// Add an entity so Get works
	store.entities[1] = &Product{BaseEntity: BaseEntity{ID: 1}, Name: "Gadget", Price: 19.99}

	// Test Get
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products/1", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("get route failed: expected %d, got %d", http.StatusOK, rr.Code)
	}

	// Test List
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("list route failed: expected %d, got %d", http.StatusOK, rr.Code)
	}

	// Test Update
	body, _ = json.Marshal(Product{Name: "Updated", Price: 29.99})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/products/1", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("update route failed: expected %d, got %d", http.StatusOK, rr.Code)
	}

	// Test Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/products/1", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("delete route failed: expected %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestHandler_RegisterRoutes_InvalidBasePath(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	mux := http.NewServeMux()
	err := h.RegisterRoutes(mux, "  ")
	if err == nil {
		t.Error("expected error for empty base path")
	}
}
