package main

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"integration/internal/config"
	"integration/internal/http-server/middleware/logger"
	"integration/internal/http-server/router"
	"integration/internal/lib/logger/sl"
	"integration/internal/storage/driver/mysql"
	"integration/internal/storage/repo"
	"log/slog"
	"net/http"
	"os"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

type RepoCloser interface {
	Close() error
}

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)
	log.Info("logger start")

	mySQLDriver, err := mysql.New(cfg.MySQLConnect)
	if err != nil {
		log.Error("Error starting pgDriver", sl.Err(err))
		os.Exit(1)
	}
	log.Info("mySQLDriver start")

	repository := repo.New(mySQLDriver)
	log.Info("repository start")

	defer func() {
		if closeErr := RepoCloser.Close(repository); closeErr != nil {
			log.Error("Error closing storage", sl.Err(closeErr))
		}
	}()

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(logger.New(log))
	r.Use(middleware.Recoverer)

	router.SetupRoutes(r, log, repository)
	log.Info("router start")

	srv := &http.Server{
		Addr:         cfg.HttpServer.Address,
		Handler:      r,
		ReadTimeout:  cfg.HttpServer.Timeout,
		WriteTimeout: cfg.HttpServer.Timeout,
		IdleTimeout:  cfg.HttpServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("Error starting http server", sl.Err(err))
	}

	log.Error("Server shutdown")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envDev:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
