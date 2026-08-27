// Command server runs the Absensi Digital PT Surya Inti Gas HTTP API.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/suryaintigas/absensi-backend/internal/config"
	"github.com/suryaintigas/absensi-backend/internal/database"
	"github.com/suryaintigas/absensi-backend/internal/health"
	"github.com/suryaintigas/absensi-backend/internal/middleware"
	"github.com/suryaintigas/absensi-backend/pkg/logger"
)

func main() {
	// .env is optional: in Docker/production, real environment variables are
	// injected by the container runtime and no file is present.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("startup_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, cfg.LogLevel)
	log.Info("starting_server", slog.String("env", cfg.AppEnv), slog.String("port", cfg.AppPort))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database_connection_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("database_connected")

	if cfg.AutoMigrate {
		migrationsPath, mErr := filepath.Abs("migrations")
		if mErr != nil {
			log.Error("migrations_path_resolution_failed", slog.String("error", mErr.Error()))
			os.Exit(1)
		}
		if mErr := database.MigrateUp(cfg.DatabaseURL, migrationsPath); mErr != nil {
			log.Error("migration_failed", slog.String("error", mErr.Error()))
			os.Exit(1)
		}
		log.Info("migrations_applied")
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.RequestLogger(log),
		middleware.Recovery(log),
		middleware.CORS(cfg.AllowedOrigins),
	)

	healthHandler := health.NewHandler(pool)
	router.GET("/health", healthHandler.Check)

	// Future API route groups (auth, employees, attendance, ...) are
	// registered here under router.Group("/api/v1") as each phase lands.

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("server_listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server_failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutdown_signal_received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful_shutdown_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("server_stopped_gracefully")
}
