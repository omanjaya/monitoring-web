package middleware

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

// resourceIDPattern matches a trailing numeric ID in a URL path segment
var resourceIDPattern = regexp.MustCompile(`/(\d+)(?:/[a-z-]+)?$`)

// AuditMiddleware logs POST, PUT, and DELETE requests to /api/* endpoints asynchronously.
func AuditMiddleware(auditRepo *mysql.AuditRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only audit mutating methods
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "DELETE" {
			c.Next()
			return
		}

		// Only audit /api/ routes
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		// Process request first
		c.Next()

		// Extract user info from context (set by auth middleware)
		userID := GetUserID(c)
		username := ""
		if claims := GetUserClaims(c); claims != nil {
			username = claims.Username
		}

		// Determine action and resource type from path
		action, resourceType := parseActionAndResource(method, path)

		// Extract resource ID from path if present
		var resourceID domain.NullInt64
		if matches := resourceIDPattern.FindStringSubmatch(path); len(matches) > 1 {
			if id, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
				resourceID = domain.NewNullInt64(id)
			}
		}

		// Build details
		details, _ := json.Marshal(map[string]interface{}{
			"method":      method,
			"path":        path,
			"status_code": c.Writer.Status(),
		})

		auditLog := &domain.AuditLog{
			UserID:       domain.NewNullInt64If(userID, userID > 0),
			Username:     domain.NewNullStringIf(username, username != ""),
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Details:      details,
			IPAddress:    domain.NewNullString(c.ClientIP()),
			UserAgent:    domain.NewNullStringIf(c.Request.UserAgent(), c.Request.UserAgent() != ""),
			CreatedAt:    time.Now(),
		}

		// Log asynchronously so it doesn't slow down the response
		go func(log *domain.AuditLog) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := auditRepo.Create(ctx, log); err != nil {
				logger.Error().Err(err).
					Str("action", log.Action).
					Str("resource_type", log.ResourceType).
					Msg("Failed to write audit log")
			}
		}(auditLog)
	}
}

// parseActionAndResource determines the action and resource_type from the HTTP method and URL path.
// Examples:
//
//	POST   /api/websites        -> action: "create",  resource_type: "website"
//	PUT    /api/websites/5      -> action: "update",  resource_type: "website"
//	DELETE /api/websites/5      -> action: "delete",  resource_type: "website"
//	POST   /api/auth/login      -> action: "login",   resource_type: "auth"
//	POST   /api/alerts/5/acknowledge -> action: "acknowledge", resource_type: "alert"
func parseActionAndResource(method, path string) (action, resourceType string) {
	// Remove /api/ prefix and split
	trimmed := strings.TrimPrefix(path, "/api/")
	parts := strings.Split(trimmed, "/")

	if len(parts) == 0 {
		return strings.ToLower(method), "unknown"
	}

	// The first segment is typically the resource type
	resourceType = singularize(parts[0])

	// Determine action based on method and path structure
	switch method {
	case "POST":
		// Check for sub-action like /alerts/5/acknowledge or /auth/login
		if len(parts) >= 3 {
			// e.g. /alerts/5/acknowledge -> action: "acknowledge"
			lastPart := parts[len(parts)-1]
			if _, err := strconv.ParseInt(lastPart, 10, 64); err != nil {
				// Last part is not a number, so it's a sub-action
				action = lastPart
				return action, resourceType
			}
		}
		if len(parts) >= 2 {
			// e.g. /auth/login -> action: "login"
			if _, err := strconv.ParseInt(parts[1], 10, 64); err != nil {
				// Second part is not a number
				// Could be /auth/login or /websites/bulk
				action = parts[1]
				return action, resourceType
			}
		}
		action = "create"
	case "PUT":
		action = "update"
	case "DELETE":
		action = "delete"
	default:
		action = strings.ToLower(method)
	}

	return action, resourceType
}

// singularize naively converts a plural resource name to singular.
func singularize(s string) string {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "ses") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}
