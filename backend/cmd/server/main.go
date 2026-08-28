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
	"github.com/suryaintigas/absensi-backend/internal/auditlog"
	"github.com/suryaintigas/absensi-backend/internal/auth"
	"github.com/suryaintigas/absensi-backend/internal/config"
	"github.com/suryaintigas/absensi-backend/internal/database"
	"github.com/suryaintigas/absensi-backend/internal/department"
	"github.com/suryaintigas/absensi-backend/internal/device"
	"github.com/suryaintigas/absensi-backend/internal/employee"
	"github.com/suryaintigas/absensi-backend/internal/faceprofile"
	"github.com/suryaintigas/absensi-backend/internal/health"
	"github.com/suryaintigas/absensi-backend/internal/middleware"
	"github.com/suryaintigas/absensi-backend/internal/position"
	"github.com/suryaintigas/absensi-backend/internal/schedule"
	"github.com/suryaintigas/absensi-backend/internal/shift"
	"github.com/suryaintigas/absensi-backend/internal/user"
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

	auditService := auditlog.NewService(auditlog.NewPostgresRepository(pool))

	authRepo := auth.NewPostgresRepository(pool)
	authService := auth.NewService(authRepo, jwtManager, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, auditService)
	authHandler := auth.NewHandler(authService)

	v1 := router.Group("/api/v1")

	authGroup := v1.Group("/auth")
	{
		// The brute-force guard belongs only on the two endpoints an
		// attacker can actually use to guess a credential (email+password,
		// or a stolen/guessed refresh token) — not on every route in this
		// group. /me requires an already-valid access token (guessing one
		// isn't a meaningful attack), and is called on every page load by
		// the dashboard's AuthProvider; sharing login's 10/min budget with
		// it meant a real user opening a few tabs, or refreshing a few
		// pages, could get rate-limited out of their own session.
		loginRateLimit := middleware.RateLimit(10, time.Minute)
		authGroup.POST("/login", loginRateLimit, authHandler.Login)
		authGroup.POST("/refresh", loginRateLimit, authHandler.Refresh)
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
	// Public: the tablet verifies itself by device_code, not a JWT — see
	// VerifyByCode's doc comment. Phase 5's app calls this on launch.
	v1.GET("/devices/verify/:code", middleware.RateLimit(30, time.Minute), deviceHandler.VerifyByCode)

	faceProfileHandler := faceprofile.NewHandler(faceprofile.NewService(faceprofile.NewPostgresRepository(pool)), deviceRepo)
	// Enrollment is an HR/Admin action on the tablet's camera (JWT-gated,
	// same roles as managing the employee record itself).
	employeeGroup.PUT("/:id/face-profile", adminOrHR, faceProfileHandler.Enroll)
	// Sync is public/device-gated like attendance, so every registered
	// tablet can download every employee's feature vector and recognize
	// faces entirely on-device, including while offline.
	v1.GET("/face-profiles/sync", middleware.RateLimit(10, time.Minute), faceProfileHandler.Sync)

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

	// --- Phase 6 support: dashboard account management + audit trail -------
	// User management is deliberately SUPER_ADMIN-only (not adminOnly): an
	// ADMIN being able to create or promote other ADMIN/SUPER_ADMIN accounts
	// would be a privilege-escalation path, unlike master data where
	// SUPER_ADMIN and ADMIN are treated as equally trusted.
	superAdminOnly := middleware.RequireRole(rbac.SuperAdmin)

	userHandler := user.NewHandler(user.NewService(user.NewPostgresRepository(pool), auditService))
	userGroup := authed.Group("/users")
	{
		userGroup.GET("", superAdminOnly, userHandler.List)
		userGroup.GET("/:id", superAdminOnly, userHandler.Get)
		userGroup.POST("", superAdminOnly, userHandler.Create)
		userGroup.PUT("/:id", superAdminOnly, userHandler.Update)
		userGroup.POST("/:id/reset-password", superAdminOnly, userHandler.ResetPassword)
		userGroup.DELETE("/:id", superAdminOnly, userHandler.Delete)
	}

	// Audit trail is read-only via HTTP — entries are only ever written by
	// auditlog.Service.Record from within another module (see auth.Service.
	// Login and internal/user's Service for the modules wired up so far).
	// SUPER_ADMIN-only: it can reveal who else has access to what, which
	// ADMIN/HR/MANAGEMENT don't need day-to-day.
	auditLogHandler := auditlog.NewHandler(auditService)
	authed.GET("/audit-logs", superAdminOnly, auditLogHandler.List)

	// Future route groups (reports, ...) are registered here under v1 as
	// each phase lands.

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
