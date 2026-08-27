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

	"github.com/suryaintigas/absensi-backend/internal/attendance"
	"github.com/suryaintigas/absensi-backend/internal/auth"
	"github.com/suryaintigas/absensi-backend/internal/config"
	"github.com/suryaintigas/absensi-backend/internal/database"
	"github.com/suryaintigas/absensi-backend/internal/department"
	"github.com/suryaintigas/absensi-backend/internal/device"
	"github.com/suryaintigas/absensi-backend/internal/employee"
	"github.com/suryaintigas/absensi-backend/internal/health"
	"github.com/suryaintigas/absensi-backend/internal/middleware"
	"github.com/suryaintigas/absensi-backend/internal/position"
	"github.com/suryaintigas/absensi-backend/internal/schedule"
	"github.com/suryaintigas/absensi-backend/internal/shift"
	"github.com/suryaintigas/absensi-backend/pkg/jwt"
	"github.com/suryaintigas/absensi-backend/pkg/logger"
	"github.com/suryaintigas/absensi-backend/pkg/rbac"
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
	log.Info("starting_server", slog.String("port", cfg.AppPort))

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

	jwtManager := jwt.NewManager(cfg.JWTSecret)

	authRepo := auth.NewPostgresRepository(pool)
	authService := auth.NewService(authRepo, jwtManager, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authHandler := auth.NewHandler(authService)

	v1 := router.Group("/api/v1")

	authGroup := v1.Group("/auth")
	authGroup.Use(middleware.RateLimit(10, time.Minute)) // brute-force guard on login/refresh
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
		authGroup.POST("/logout", authHandler.Logout)
		authGroup.GET("/me", middleware.AuthRequired(jwtManager), authHandler.Me)
	}

	// --- Phase 3: master data ---------------------------------------------
	// Every route below requires a valid access token. Read (list/detail) is
	// open to any authenticated role; mutations are restricted per the
	// permission matrix in README.md's Master Data section.
	authed := v1.Group("")
	authed.Use(middleware.AuthRequired(jwtManager))

	adminOnly := middleware.RequireRole(rbac.SuperAdmin, rbac.Admin)
	adminOrHR := middleware.RequireRole(rbac.SuperAdmin, rbac.Admin, rbac.HR)

	deptHandler := department.NewHandler(department.NewService(department.NewPostgresRepository(pool)))
	deptGroup := authed.Group("/departments")
	{
		deptGroup.GET("", deptHandler.List)
		deptGroup.GET("/:id", deptHandler.Get)
		deptGroup.POST("", adminOnly, deptHandler.Create)
		deptGroup.PUT("/:id", adminOnly, deptHandler.Update)
		deptGroup.DELETE("/:id", adminOnly, deptHandler.Delete)
	}

	posHandler := position.NewHandler(position.NewService(position.NewPostgresRepository(pool)))
	posGroup := authed.Group("/positions")
	{
		posGroup.GET("", posHandler.List)
		posGroup.GET("/:id", posHandler.Get)
		posGroup.POST("", adminOnly, posHandler.Create)
		posGroup.PUT("/:id", adminOnly, posHandler.Update)
		posGroup.DELETE("/:id", adminOnly, posHandler.Delete)
	}

	// Repositories are named so Phase 4's attendance service can reuse the
	// same instances to re-validate employee/device/shift/schedule,
	// instead of re-querying through those modules' own services.
	shiftRepo := shift.NewPostgresRepository(pool)
	shiftHandler := shift.NewHandler(shift.NewService(shiftRepo))
	shiftGroup := authed.Group("/shifts")
	{
		shiftGroup.GET("", shiftHandler.List)
		shiftGroup.GET("/:id", shiftHandler.Get)
		shiftGroup.POST("", adminOrHR, shiftHandler.Create)
		shiftGroup.PUT("/:id", adminOrHR, shiftHandler.Update)
		shiftGroup.DELETE("/:id", adminOrHR, shiftHandler.Delete)
	}

	employeeRepo := employee.NewPostgresRepository(pool)
	employeeHandler := employee.NewHandler(employee.NewService(employeeRepo))
	employeeGroup := authed.Group("/employees")
	{
		employeeGroup.GET("", employeeHandler.List)
		employeeGroup.GET("/:id", employeeHandler.Get)
		employeeGroup.POST("", adminOrHR, employeeHandler.Create)
		employeeGroup.PUT("/:id", adminOrHR, employeeHandler.Update)
		employeeGroup.DELETE("/:id", adminOrHR, employeeHandler.Delete)
	}

	scheduleRepo := schedule.NewPostgresRepository(pool)
	scheduleHandler := schedule.NewHandler(schedule.NewService(scheduleRepo))
	scheduleGroup := authed.Group("/schedules")
	{
		scheduleGroup.GET("", scheduleHandler.List)
		scheduleGroup.GET("/:id", scheduleHandler.Get)
		scheduleGroup.POST("", adminOrHR, scheduleHandler.Create)
		scheduleGroup.PUT("/:id", adminOrHR, scheduleHandler.Update)
		scheduleGroup.DELETE("/:id", adminOrHR, scheduleHandler.Delete)
	}

	deviceRepo := device.NewPostgresRepository(pool)
	deviceHandler := device.NewHandler(device.NewService(deviceRepo))
	deviceGroup := authed.Group("/devices")
	{
		deviceGroup.GET("", deviceHandler.List)
		deviceGroup.GET("/:id", deviceHandler.Get)
		deviceGroup.POST("/register", adminOnly, deviceHandler.Register)
		deviceGroup.PUT("/:id", adminOnly, deviceHandler.Update)
		deviceGroup.DELETE("/:id", adminOnly, deviceHandler.Delete)
	}

	// --- Phase 4: attendance ------------------------------------------------
	// Check-in/check-out are deliberately NOT behind AuthRequired: the
	// tablet has no dashboard login. The registered, active device_code it
	// presents is the trust boundary instead (see internal/attendance's
	// package doc). List/detail (attendance history) are dashboard-facing
	// and require the same authentication as every other master data read.
	attendanceHandler := attendance.NewHandler(attendance.NewService(
		attendance.NewPostgresRepository(pool), employeeRepo, deviceRepo, shiftRepo, scheduleRepo,
	))
	attendancePublic := v1.Group("/attendance")
	attendancePublic.Use(middleware.RateLimit(60, time.Minute)) // generous: many employees clock in around the same few minutes
	{
		attendancePublic.POST("/check-in", attendanceHandler.CheckIn)
		attendancePublic.POST("/check-out", attendanceHandler.CheckOut)
	}
	attendanceGroup := authed.Group("/attendance")
	{
		attendanceGroup.GET("", attendanceHandler.List)
		attendanceGroup.GET("/:id", attendanceHandler.Get)
	}

	// Future route groups (reports, audit logs, ...) are registered here
	// under v1 as each phase lands.

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
