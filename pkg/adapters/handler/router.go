package handler

import (
	"encoding/json"
	"net/http"

	"github.com/wadjakorntonsri/go-url-shortener/pkg/config"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/ports"
)

// NewRouter creates and configures the main application router
func NewRouter(cfg *config.Config, service ports.LinkService, collectionService ports.CollectionService) http.Handler {
	// Initialize Handlers
	h := NewHTTPHandler(service)
	ch := NewCollectionHandler(collectionService)

	// Initialize Middleware
	mw := NewMiddleware(cfg)

	// Initialize Auth Handler
	authHandler := NewAuthHandler(cfg)

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
	mux.HandleFunc("GET /u/{slug}", ch.GetPublicCollection)
	mux.HandleFunc("GET /auth/google/login", authHandler.Login)
	mux.HandleFunc("GET /auth/google/callback", authHandler.Callback)
	mux.HandleFunc("GET /auth/logout", authHandler.Logout)

	// Protected Routes (API & Dashboard)
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("POST /api/v1/links", h.Create)
	protectedMux.HandleFunc("GET /api/v1/links", h.List)
	protectedMux.HandleFunc("GET /api/v1/links/{id}/stats", h.Stats)
	protectedMux.HandleFunc("GET /api/v1/dashboard", h.Dashboard)
	protectedMux.HandleFunc("PUT /api/v1/links/{id}", h.Update)
	protectedMux.HandleFunc("DELETE /api/v1/links/{id}", h.Delete)

	// Collection Routes
	protectedMux.HandleFunc("POST /api/v1/collections", ch.CreateCollection)
	protectedMux.HandleFunc("GET /api/v1/collections", ch.ListCollections)
	protectedMux.HandleFunc("GET /api/v1/collections/{id}", ch.GetCollection)
	protectedMux.HandleFunc("PUT /api/v1/collections/{id}", ch.UpdateCollection)
	protectedMux.HandleFunc("DELETE /api/v1/collections/{id}", ch.DeleteCollection)
	protectedMux.HandleFunc("POST /api/v1/collections/{id}/links", ch.AddLink)
	protectedMux.HandleFunc("DELETE /api/v1/collections/{id}/links/{linkID}", ch.RemoveLink)

	// Apply Middleware to Protected Routes
	// Note: We match /api/v1/ to capture all API requests.
	// Since protectedMux contains the full paths, this works for dispatching.
	mux.Handle("/api/v1/", mw.AuthMiddleware(protectedMux))

	return mux
}
