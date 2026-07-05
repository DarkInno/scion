package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// auth is the copied Scion auth module.
// In your real project, replace the import path:
//
//	import "yourproject/internal/auth"
//
// For this standalone example to compile, you would need to:
// 1. Copy registry/auth/src/go/ into a local directory
// 2. Replace import paths in go.mod
// 3. Run `go mod tidy`

func main() {
	_ = godotenv.Load()

	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "this-is-a-32-character-secret-key-for-dev")
	}
	if os.Getenv("DB_URL") == "" {
		os.Setenv("DB_URL", "sqlite://dev.db")
	}

	// cfg, err := auth.LoadConfig()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// store := NewInMemoryStore()
	// handler := auth.NewHandler(store, cfg)
	//
	// mux := http.NewServeMux()
	// handler.RegisterRoutes(mux)
	//
	// log.Println("Server running on :8080")
	// log.Fatal(http.ListenAndServe(":8080", mux))

	log.Println("This is a template example. See registry/auth/src/go/ for the actual source code.")
	log.Println("Copy the source files into your project and implement the UserStore interface.")
}
