package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/suryaintigas/absensi-backend/pkg/response"
)

// RateLimit is a simple in-process, per-client-IP fixed-window limiter used
// to slow down brute-force login attempts. It is intentionally
// dependency-free (no Redis) to match this project's "modular monolith, no
// extra infrastructure for v1" principle — if the backend is ever scaled to
// multiple instances behind a load balancer, replace this with a shared
// store (e.g. Redis) so limits apply across instances, not per-instance.
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	type bucket struct {
		count   int
		resetAt time.Time
	}

	var (
		mu      sync.Mutex
		buckets = make(map[string]*bucket)
	)

	// Periodically drop expired buckets so long-running processes don't
	// accumulate one entry per distinct client IP forever.
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			mu.Lock()
			for key, b := range buckets {
				if now.After(b.resetAt) {
					delete(buckets, key)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()

		mu.Lock()
		b, exists := buckets[key]
		if !exists || now.After(b.resetAt) {
			b = &bucket{resetAt: now.Add(window)}
			buckets[key] = b
		}
		b.count++
		blocked := b.count > maxRequests
		mu.Unlock()

		if blocked {
			response.Fail(c, http.StatusTooManyRequests, "Terlalu banyak percobaan. Silakan coba lagi nanti.", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
