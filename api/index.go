package handler

import (
	"net/http"

	"github.com/wadjakorntonsri/go-url-shortener/pkg/adapters/handler"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/adapters/repository/sqlite"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/config"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/core/services"
)

var mux http.Handler

func init() {
	cfg := config.Load()

	// Note: On Vercel, db.sqlite is ephemeral unless using a remote SQL/Turso URL in DATABASE_URL
	repo, err := sqlite.NewSQLiteRepository(cfg.DatabaseURL)
	if err != nil {
		// Log but don't fatal, let internal error happen on request if crucial
		panic(err)
	}

	service := services.NewLinkService(repo)
	mux = handler.NewRouter(cfg, service)
}

// Handler is the entrypoint for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	mux.ServeHTTP(w, r)
}
