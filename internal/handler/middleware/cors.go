package middleware

import (
	"net/http"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a permissive CORS handler (backward compatible).
// Prefer CORSMiddlewareWithConfig for production use.
func CORSMiddleware() gin.HandlerFunc {
	return CORSMiddlewareWithConfig(nil)
}

// CORSMiddlewareWithConfig returns a CORS handler that restricts origins
// based on configuration. When cfg is nil or AllowedOrigins contains "*",
// all origins are allowed (suitable for development).
func CORSMiddlewareWithConfig(cfg *config.Config) gin.HandlerFunc {
	// Pre-compute allowed origins set for fast lookup.
	allowedOrigins := []string{"*"}
	if cfg != nil && len(cfg.Server.AllowedOrigins) > 0 {
		allowedOrigins = cfg.Server.AllowedOrigins
	}

	wildcard := contains(allowedOrigins, "*")

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if wildcard {
			// Wildcard mode: allow any origin but never combine with credentials.
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && contains(allowedOrigins, origin) {
			// Explicit origin match: echo it back and allow credentials.
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		} else {
			// Origin not in allow-list; do not set any CORS headers.
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Expose-Headers", "Content-Length")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
