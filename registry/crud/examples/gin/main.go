package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Product represents a sample entity.
type Product struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InMemoryStore is a minimal example store. Replace with your real DB layer
// (GORM, sqlx, pgx, etc.) in production.
type InMemoryStore struct {
	items  map[uint]*Product
	nextID uint
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: make(map[uint]*Product)}
}

func (s *InMemoryStore) Create(entity *Product) (*Product, error) {
	s.nextID++
	entity.ID = s.nextID
	entity.CreatedAt = time.Now()
	entity.UpdatedAt = time.Now()
	s.items[entity.ID] = entity
	return entity, nil
}

func (s *InMemoryStore) GetByID(id uint) (*Product, error) {
	p, ok := s.items[id]
	if !ok {
		return nil, http.ErrNoLocation // placeholder error
	}
	return p, nil
}

func (s *InMemoryStore) List(params interface{}) ([]Product, int64, error) {
	var results []Product
	for _, p := range s.items {
		results = append(results, *p)
	}
	return results, int64(len(results)), nil
}

func (s *InMemoryStore) Update(id uint, entity *Product) (*Product, error) {
	if _, ok := s.items[id]; !ok {
		return nil, http.ErrNoLocation
	}
	entity.ID = id
	entity.UpdatedAt = time.Now()
	s.items[id] = entity
	return entity, nil
}

func (s *InMemoryStore) Delete(id uint) error {
	delete(s.items, id)
	return nil
}

func main() {
	_ = godotenv.Load()

	if os.Getenv("DB_URL") == "" {
		os.Setenv("DB_URL", "sqlite://dev.db")
	}

	// cfg, err := crud.LoadConfig()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// store := NewInMemoryStore()
	// handler := crud.NewHandler[Product](store, cfg)
	// handler.WithSortValidator(func(field string) bool {
	// 	return field == "name" || field == "price" || field == "created_at"
	// })
	// handler.WithFilterFields(map[string]bool{"name": true})
	//
	// mux := http.NewServeMux()
	// handler.RegisterRoutes(mux, "/api/v1/products")
	//
	// log.Println("Server running on :8080")
	// log.Fatal(http.ListenAndServe(":8080", mux))

	log.Println("This is a template example. See registry/crud/src/go/ for the actual source code.")
	log.Println("Copy the source files into your project and implement the EntityStore[T] interface.")
}
