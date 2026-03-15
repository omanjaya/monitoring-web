package domain

import (
	"time"
)

// User represents a system user
type User struct {
	ID           int64          `db:"id" json:"id"`
	Username     string         `db:"username" json:"username"`
	Email        string         `db:"email" json:"email"`
	PasswordHash string         `db:"password_hash" json:"-"` // Never expose
	FullName     string         `db:"full_name" json:"full_name"`
	Phone        NullString `db:"phone" json:"phone,omitempty"`
	Role         string         `db:"role" json:"role"`
	IsActive     bool           `db:"is_active" json:"is_active"`
	LastLoginAt  NullTime   `db:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
}

// UserLogin is the input for login
type UserLogin struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserCreate is the input for creating a user
type UserCreate struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone"`
}

// UserUpdate is the input for updating a user
type UserUpdate struct {
	Email    *string `json:"email" binding:"omitempty,email"`
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
	IsActive *bool   `json:"is_active"`
}

// PasswordChange is the input for changing password
type PasswordChange struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// Keyword represents a detection keyword
type Keyword struct {
	ID        int64     `db:"id" json:"id"`
	Keyword   string    `db:"keyword" json:"keyword"`
	Category  string    `db:"category" json:"category"` // gambling, defacement, malware
	IsRegex   bool      `db:"is_regex" json:"is_regex"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	Weight    int       `db:"weight" json:"weight"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Setting is now defined in settings.go with SettingType enum
