package crud

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// Product is a sample entity for testing generics.
type Product struct {
	BaseEntity
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// mockEntityStore is a test double for EntityStore.
type mockEntityStore[T any] struct {
	entities  map[uint]*T
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
	listData  []T
	listTotal int64
	lastList  ListParams
}

func newMockEntityStore[T any]() *mockEntityStore[T] {
	return &mockEntityStore[T]{entities: make(map[uint]*T)}
}

func (m *mockEntityStore[T]) Create(entity *T) (*T, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return entity, nil
}

func (m *mockEntityStore[T]) GetByID(id uint) (*T, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	e, ok := m.entities[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return e, nil
}

func (m *mockEntityStore[T]) List(params ListParams) ([]T, int64, error) {
	m.lastList = params
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	if m.listData != nil {
		return m.listData, m.listTotal, nil
	}
	return make([]T, 0), 0, nil
}

func (m *mockEntityStore[T]) Update(id uint, entity *T) (*T, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return entity, nil
}

func (m *mockEntityStore[T]) Delete(id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.entities, id)
	return nil
}

func TestHandler_Create(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	p := Product{Name: "Widget", Price: 9.99}
	body, _ := json.Marshal(p)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var resp Product
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Name != "Widget" {
		t.Errorf("expected name Widget, got %s", resp.Name)
	}
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte("not json")))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_Get(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	p := &Product{BaseEntity: BaseEntity{ID: 1}, Name: "Gadget", Price: 19.99}
	store.entities[1] = p

	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp Product
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
}

func TestHandler_Get_InvalidID(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/products/abc", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_Get_NotFound(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/products/999", nil)
	req.SetPathValue("id", "999")
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandler_List(t *testing.T) {
	store := newMockEntityStore[Product]()
	store.listData = []Product{
		{BaseEntity: BaseEntity{ID: 1}, Name: "A", Price: 1},
		{BaseEntity: BaseEntity{ID: 2}, Name: "B", Price: 2},
	}
	store.listTotal = 2

	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg).WithSortValidator(func(field string) bool {
		return field == "name" || field == "price"
	})

	req := httptest.NewRequest(http.MethodGet, "/products?offset=0&limit=10&sort=name", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp PaginatedResponse[Product]
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Data))
	}
	if resp.Data == nil {
		t.Error("expected data to be non-nil slice")
	}
}

func TestHandler_List_InvalidSort(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg).WithSortValidator(func(_ string) bool {
		return false // reject all
	})

	req := httptest.NewRequest(http.MethodGet, "/products?sort=evil_column", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_List_FilterAllowed(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg).WithFilterFields(map[string]bool{"status": true})

	req := httptest.NewRequest(http.MethodGet, "/products?status=active", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestHandler_List_FilterDisabledByDefault(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/products?password=secret", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if store.lastList.Filter != nil {
		t.Fatalf("expected filters to be disabled, got %#v", store.lastList.Filter)
	}
}

func TestHandler_List_FilterDisallowed(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg).WithFilterFields(map[string]bool{"status": true})

	req := httptest.NewRequest(http.MethodGet, "/products?evil=drop+table", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid filter field") {
		t.Errorf("expected filter field error, got %s", rr.Body.String())
	}
}

func TestHandler_List_RejectsTooManyFilters(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	allowed := make(map[string]bool, maxFilterCount+1)
	q := make([]string, 0, maxFilterCount+1)
	for i := 0; i < maxFilterCount+1; i++ {
		key := "f" + strconv.Itoa(i)
		allowed[key] = true
		q = append(q, key+"=x")
	}
	h := NewHandler(store, cfg).WithFilterFields(allowed)

	req := httptest.NewRequest(http.MethodGet, "/products?"+strings.Join(q, "&"), nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_List_NilSliceProtection(t *testing.T) {
	store := newMockEntityStore[Product]()
	// listData is nil, listTotal is 0

	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if strings.Contains(body, `"data":null`) {
		t.Error("expected data to be [] not null")
	}
	if !strings.Contains(body, `"data":[]`) {
		t.Error("expected data to be empty array")
	}
}

func TestHandler_List_LimitCapped(t *testing.T) {
	store := newMockEntityStore[Product]()
	store.listData = []Product{{Name: "A"}}
	store.listTotal = 1

	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 50}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/products?limit=999", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandler_Update(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	p := Product{Name: "Updated", Price: 29.99}
	body, _ := json.Marshal(p)
	req := httptest.NewRequest(http.MethodPut, "/products/1", bytes.NewReader(body))
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp Product
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Name != "Updated" {
		t.Errorf("expected name Updated, got %s", resp.Name)
	}
}

func TestHandler_Update_InvalidID(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodPut, "/products/abc", bytes.NewReader([]byte("{}")))
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_Delete(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestHandler_Delete_InvalidID(t *testing.T) {
	store := newMockEntityStore[Product]()
	cfg := &Config{DefaultPageSize: 20, MaxPageSize: 100}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodDelete, "/products/abc", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestDecodeBodyRejectsOversizedValidJSONPrefix(t *testing.T) {
	prefix := []byte(`{"name":"Widget"}`)
	body := append(prefix, bytes.Repeat([]byte(" "), maxRequestBodySize-len(prefix)+1)...)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))

	var dst Product
	if err := decodeBody(req, &dst); err == nil {
		t.Fatal("expected oversized body error")
	}
}

func TestSortFieldFromRaw(t *testing.T) {
	if sortFieldFromRaw("name") != "name" {
		t.Error("expected name")
	}
	if sortFieldFromRaw("-created_at") != "created_at" {
		t.Error("expected created_at")
	}
	if sortFieldFromRaw("") != "" {
		t.Error("expected empty")
	}
}
