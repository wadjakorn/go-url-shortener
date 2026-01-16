package main

import (
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
	collectionService := services.NewCollectionService(repo)

	// Initialize Router
	mux := handler.NewRouter(cfg, service, collectionService)

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
