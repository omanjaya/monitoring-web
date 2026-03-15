package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

type rateLimiter struct {
	mu          sync.Mutex
	entries     map[string]*rateLimitEntry
	maxRequests int
	window      time.Duration
}

func newRateLimiter(maxRequests int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		entries:     make(map[string]*rateLimitEntry),
		maxRequests: maxRequests,
		window:      window,
	}

	// Periodically clean up expired entries
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, entry := range rl.entries {
		if now.Sub(entry.windowStart) > rl.window {
			delete(rl.entries, ip)
		}
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[ip]

	if !exists || now.Sub(entry.windowStart) > rl.window {
		rl.entries[ip] = &rateLimitEntry{
			count:       1,
			windowStart: now,
		}
		return true
	}

	entry.count++
	return entry.count <= rl.maxRequests
}

func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		return xff
	}
	return c.ClientIP()
}

// RateLimitMiddleware creates a general rate limiter middleware.
// maxRequests is the maximum number of requests allowed within the given window duration per IP.
func RateLimitMiddleware(maxRequests int, window time.Duration) gin.HandlerFunc {
	limiter := newRateLimiter(maxRequests, window)

	return func(c *gin.Context) {
		ip := getClientIP(c)

		if !limiter.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}

// LoginRateLimitMiddleware returns a rate limiter pre-configured for login endpoints:
// 5 attempts per 1 minute per IP.
func LoginRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(5, time.Minute)
}
