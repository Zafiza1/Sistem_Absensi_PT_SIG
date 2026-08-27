// Package health exposes the service's liveness/readiness endpoint.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/suryaintigas/absensi-backend/pkg/response"
)

// Handler serves GET /health.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler builds a health Handler backed by the given database pool.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// Check reports overall service health plus database connectivity. It
// returns 200 when the database is reachable and 503 otherwise, so it can
// double as a Docker/orchestrator readiness probe, not just a liveness one.
func (h *Handler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "up"
	httpStatus := http.StatusOK
	if err := h.pool.Ping(ctx); err != nil {
		dbStatus = "down"
		httpStatus = http.StatusServiceUnavailable
	}

	data := gin.H{
		"status":   "ok",
		"time":     time.Now().UTC().Format(time.RFC3339),
		"database": dbStatus,
		"service":  "absensi-backend",
	}
	if httpStatus != http.StatusOK {
		data["status"] = "degraded"
	}

	response.OK(c, httpStatus, "Service health status", data)
}
