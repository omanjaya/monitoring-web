package middleware

import (
	"strings"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/gin-gonic/gin"
)

const defaultCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; " +
	"style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; " +
	"font-src 'self' https://cdnjs.cloudflare.com; " +
	"img-src 'self' data:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'"

// SecurityHeadersMiddleware returns the security headers handler with
// default CSP (backward compatible). Prefer SecurityHeadersMiddlewareWithConfig.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return SecurityHeadersMiddlewareWithConfig(nil)
}

// SecurityHeadersMiddlewareWithConfig returns a handler that sets security
// headers. The CSP policy can be overridden via cfg.Server.CSPPolicy;
// when empty the default policy is used. CSP is only applied to non-API
// routes since API JSON responses do not need it.
func SecurityHeadersMiddlewareWithConfig(cfg *config.Config) gin.HandlerFunc {
	csp := defaultCSP
	if cfg != nil && cfg.Server.CSPPolicy != "" {
		csp = cfg.Server.CSPPolicy
	}

	return func(c *gin.Context) {
		// Apply CSP only to non-API routes (HTML pages).
		if !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Content-Security-Policy", csp)
		}

		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		c.Next()
	}
}
