package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/handler/middleware"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type UserHandler struct {
	userRepo *mysql.UserRepository
}

func NewUserHandler(userRepo *mysql.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

// CreateUserInput extends UserCreate with a role field
type CreateUserInput struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

// ResetPasswordInput is the input for admin password reset
type ResetPasswordInput struct {
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ListUsers retrieves all users
// @Summary List all users
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.GetAll(c.Request.Context())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get users")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    users,
		"message": "Berhasil mengambil data user",
	})
}

// CreateUser creates a new user
// @Summary Create a new user
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body CreateUserInput true "User data"
// @Success 201 {object} map[string]interface{}
// @Router /api/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var input CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	// Check if username already exists
	existing, err := h.userRepo.GetByUsername(c.Request.Context(), input.Username)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check existing username")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal memeriksa username",
		})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Username sudah digunakan",
		})
		return
	}

	// Check if email already exists
	existing, err = h.userRepo.GetByEmail(c.Request.Context(), input.Email)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check existing email")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal memeriksa email",
		})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Email sudah digunakan",
		})
		return
	}

	// Hash password with bcrypt cost 12
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal memproses password",
		})
		return
	}

	// Set default role
	role := input.Role
	if role == "" {
		role = "admin"
	}

	userCreate := &domain.UserCreate{
		Username: input.Username,
		Email:    input.Email,
		Password: input.Password,
		FullName: input.FullName,
		Phone:    input.Phone,
	}

	id, err := h.userRepo.Create(c.Request.Context(), userCreate, string(hashedPassword), role)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat user: " + err.Error(),
		})
		return
	}

	// Fetch created user to return
	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch created user")
		c.JSON(http.StatusCreated, gin.H{
			"data":    gin.H{"id": id},
			"message": "User berhasil dibuat",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    user,
		"message": "User berhasil dibuat",
	})
}

// UpdateUser updates an existing user
// @Summary Update user details
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param input body domain.UserUpdate true "User update data"
// @Success 200 {object} map[string]interface{}
// @Router /api/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	// Check if user exists
	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data user",
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User tidak ditemukan",
		})
		return
	}

	var input domain.UserUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	// If email is being changed, check for duplicates
	if input.Email != nil && *input.Email != user.Email {
		existing, err := h.userRepo.GetByEmail(c.Request.Context(), *input.Email)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to check existing email")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Gagal memeriksa email",
			})
			return
		}
		if existing != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email sudah digunakan",
			})
			return
		}
	}

	if err := h.userRepo.Update(c.Request.Context(), id, &input); err != nil {
		logger.Error().Err(err).Msg("Failed to update user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengupdate user",
		})
		return
	}

	// Fetch updated user
	updated, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch updated user")
		c.JSON(http.StatusOK, gin.H{
			"message": "User berhasil diupdate",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    updated,
		"message": "User berhasil diupdate",
	})
}

// DeleteUser deletes a user
// @Summary Delete a user
// @Tags Users
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	// Prevent self-delete
	currentUserID := middleware.GetUserID(c)
	if currentUserID == id {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Tidak dapat menghapus akun sendiri",
		})
		return
	}

	// Check if user exists
	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data user",
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User tidak ditemukan",
		})
		return
	}

	if err := h.userRepo.Delete(c.Request.Context(), id); err != nil {
		logger.Error().Err(err).Msg("Failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menghapus user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User berhasil dihapus",
	})
}

// ResetUserPassword resets a user's password (admin action)
// @Summary Reset user password
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param input body ResetPasswordInput true "New password"
// @Success 200 {object} map[string]interface{}
// @Router /api/users/{id}/reset-password [post]
func (h *UserHandler) ResetUserPassword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	// Check if user exists
	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data user",
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User tidak ditemukan",
		})
		return
	}

	var input ResetPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password baru wajib diisi (minimal 8 karakter)",
		})
		return
	}

	// Hash new password with bcrypt cost 12
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), 12)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal memproses password",
		})
		return
	}

	if err := h.userRepo.UpdatePassword(c.Request.Context(), id, string(hashedPassword)); err != nil {
		logger.Error().Err(err).Msg("Failed to reset password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mereset password",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password user berhasil direset",
	})
}
