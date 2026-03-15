package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/service/auth"
)

const (
	AuthorizationHeader = "Authorization"
	UserContextKey      = "user"
	UserIDContextKey    = "user_id"
)

func AuthMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			return
		}

		// Check Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			return
		}

		// Validate token
		claims, err := authService.ValidateToken(parts[1])
		if err != nil {
			status := http.StatusUnauthorized
			message := "Token tidak valid"

			if err == auth.ErrTokenExpired {
				message = "Token sudah expired"
			}

			c.AbortWithStatusJSON(status, gin.H{
				"error": message,
			})
			return
		}

		// Set user info in context
		c.Set(UserContextKey, claims)
		c.Set(UserIDContextKey, claims.UserID)

		c.Next()
	}
}

// GetUserClaims retrieves user claims from context
func GetUserClaims(c *gin.Context) *auth.Claims {
	claims, exists := c.Get(UserContextKey)
	if !exists {
		return nil
	}
	return claims.(*auth.Claims)
}

// GetUserID retrieves user ID from context
func GetUserID(c *gin.Context) int64 {
	userID, exists := c.Get(UserIDContextKey)
	if !exists {
		return 0
	}
	return userID.(int64)
}

// OptionalAuthMiddleware allows requests with or without authentication
func OptionalAuthMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.Next()
			return
		}

		claims, err := authService.ValidateToken(parts[1])
		if err == nil {
			c.Set(UserContextKey, claims)
			c.Set(UserIDContextKey, claims.UserID)
		}

		c.Next()
	}
}
