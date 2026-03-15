package domain

import (
	"time"
)

// SecurityHeaderCheck represents a security headers check result
type SecurityHeaderCheck struct {
	ID          int64          `db:"id" json:"id"`
	WebsiteID   int64          `db:"website_id" json:"website_id"`
	Score       int            `db:"score" json:"score"` // 0-100
	Grade       string         `db:"grade" json:"grade"` // A+, A, B, C, D, F
	Headers     NullString `db:"headers" json:"headers,omitempty"` // JSON of HeaderResult
	Findings    NullString `db:"findings" json:"findings,omitempty"` // JSON of SecurityFinding
	CheckedAt   time.Time      `db:"checked_at" json:"checked_at"`
}

// HeaderResult represents the check result for each header
type HeaderResult struct {
	Name        string `json:"name"`
	Present     bool   `json:"present"`
	Value       string `json:"value,omitempty"`
	Expected    bool   `json:"expected"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // high, medium, low
	Points      int    `json:"points"` // Points earned for this header
	MaxPoints   int    `json:"max_points"`
}

// SecurityFinding represents a security issue found
type SecurityFinding struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"` // critical, high, medium, low, info
	Title       string `json:"title"`
	Description string `json:"description"`
	Remedy      string `json:"remedy"`
}

// SecurityHeadersSummary represents summary of security headers for a website
type SecurityHeadersSummary struct {
	WebsiteID     int64     `json:"website_id"`
	WebsiteName   string    `json:"website_name"`
	WebsiteURL    string    `json:"website_url"`
	Score         int       `json:"score"`
	Grade         string    `json:"grade"`
	LastCheckedAt time.Time `json:"last_checked_at"`
}

// SecurityStats represents overall security statistics
type SecurityStats struct {
	TotalWebsites     int                   `json:"total_websites"`
	AverageScore      float64               `json:"average_score"`
	GradeDistribution map[string]int        `json:"grade_distribution"`
	CommonIssues      []CommonSecurityIssue `json:"common_issues"`
	SSLValid          int                   `json:"ssl_valid"`
	SSLExpiringSoon   int                   `json:"ssl_expiring_soon"`
	MissingHeaders    int                   `json:"missing_headers"`
}

// CommonSecurityIssue represents a commonly found security issue
type CommonSecurityIssue struct {
	HeaderName string `json:"header_name"`
	MissingCount int `json:"missing_count"`
	Percentage float64 `json:"percentage"`
}

// Required security headers with their configurations
var SecurityHeaders = []struct {
	Name        string
	Required    bool
	Weight      int // max points
	Description string
	Impact      string
}{
	{
		Name:        "Content-Security-Policy",
		Required:    true,
		Weight:      20,
		Description: "Prevents XSS attacks by restricting resources the browser can load",
		Impact:      "high",
	},
	{
		Name:        "Strict-Transport-Security",
		Required:    true,
		Weight:      15,
		Description: "Enforces HTTPS connections to prevent man-in-the-middle attacks",
		Impact:      "high",
	},
	{
		Name:        "X-Frame-Options",
		Required:    true,
		Weight:      15,
		Description: "Prevents clickjacking attacks by controlling iframe embedding",
		Impact:      "high",
	},
	{
		Name:        "X-Content-Type-Options",
		Required:    true,
		Weight:      10,
		Description: "Prevents MIME type sniffing attacks",
		Impact:      "medium",
	},
	{
		Name:        "X-XSS-Protection",
		Required:    false,
		Weight:      5,
		Description: "Legacy XSS protection for older browsers (deprecated in modern browsers)",
		Impact:      "low",
	},
	{
		Name:        "Referrer-Policy",
		Required:    true,
		Weight:      10,
		Description: "Controls how much referrer information should be sent",
		Impact:      "medium",
	},
	{
		Name:        "Permissions-Policy",
		Required:    true,
		Weight:      10,
		Description: "Controls which browser features can be used",
		Impact:      "medium",
	},
	{
		Name:        "X-Permitted-Cross-Domain-Policies",
		Required:    false,
		Weight:      5,
		Description: "Restricts Adobe Flash and PDF readers from loading content",
		Impact:      "low",
	},
	{
		Name:        "Cache-Control",
		Required:    false,
		Weight:      5,
		Description: "Controls caching behavior for sensitive data",
		Impact:      "low",
	},
	{
		Name:        "Cross-Origin-Embedder-Policy",
		Required:    false,
		Weight:      5,
		Description: "Prevents loading cross-origin resources without explicit permission",
		Impact:      "medium",
	},
	{
		Name:        "Cross-Origin-Opener-Policy",
		Required:    false,
		Weight:      5,
		Description: "Isolates browsing context to prevent cross-origin attacks",
		Impact:      "medium",
	},
	{
		Name:        "Cross-Origin-Resource-Policy",
		Required:    false,
		Weight:      5,
		Description: "Controls which origins can load the resource to prevent cross-origin data leaks",
		Impact:      "medium",
	},
}

// CalculateGrade returns the grade based on score
func CalculateGrade(score int) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 85:
		return "A"
	case score >= 70:
		return "B"
	case score >= 55:
		return "C"
	case score >= 40:
		return "D"
	default:
		return "F"
	}
}
