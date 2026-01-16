package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/wadjakorntonsri/go-url-shortener/pkg/adapters/handler"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/adapters/repository/sqlite"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/config"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/core/services"
)

func main() {
	cfg := config.Load()

	// Initialize Repository
	repo, err := sqlite.NewSQLiteRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Service
	service := services.NewLinkService(repo)

	// Initialize Handler
	h := handler.NewHTTPHandler(service)

	// Initialize Middleware
	mw := handler.NewMiddleware(cfg)

	// Initialize Auth Handler
	authHandler := handler.NewAuthHandler(cfg)

	// Setup Router
	mux := http.NewServeMux()

	// Public Routes
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		res := map[string]string{
			"message": "ok",
		}
		_ = json.NewEncoder(w).Encode(&res)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /open/{short_code}", h.Redirect)
	mux.HandleFunc("GET /auth/google/login", authHandler.Login)
	mux.HandleFunc("GET /auth/google/callback", authHandler.Callback)
	mux.HandleFunc("GET /app/logout", authHandler.Logout)

	// Protected Routes (API & Dashboard)
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("POST /api/v1/links", h.Create)
	protectedMux.HandleFunc("GET /api/v1/links", h.List)
	protectedMux.HandleFunc("GET /api/v1/links/{id}/stats", h.Stats)
	protectedMux.HandleFunc("GET /api/v1/dashboard", h.Dashboard)
	protectedMux.HandleFunc("PUT /api/v1/links/{id}", h.Update)
	protectedMux.HandleFunc("DELETE /api/v1/links/{id}", h.Delete)

	// Apply Middleware to Protected Routes
	// Note: We match /api/v1/ to capture all API requests.
	// Since protectedMux contains the full paths, this works for dispatching.
	mux.Handle("/api/v1/", mw.AuthMiddleware(protectedMux))

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Server starting on port %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
