package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"integration/internal/http-server/handlers/reservoir"
	"integration/internal/storage/repo"
	"log/slog"
)

func SetupRoutes(router *chi.Mux, log *slog.Logger, mysql *repo.Repo) {
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.Get("/ac-wh-1000xm5/reservoir/{reservoirId}", reservoir.New(log, mysql))
}
