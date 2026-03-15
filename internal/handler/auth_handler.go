package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/handler/middleware"
	"github.com/diskominfos-bali/monitoring-website/internal/service/auth"
)

type AuthHandler struct {
	authService *auth.Service
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login handles user login
// @Summary User login
// @Tags Auth
// @Accept json
// @Produce json
// @Param input body auth.LoginInput true "Login credentials"
// @Success 200 {object} auth.LoginResponse
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input auth.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Username dan password wajib diisi",
		})
		return
	}

	response, err := h.authService.Login(c.Request.Context(), &input)
	if err != nil {
		status := http.StatusUnauthorized
		if err == auth.ErrInvalidCredentials {
			c.JSON(status, gin.H{
				"error": "Username atau password salah",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Terjadi kesalahan saat login",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetMe retrieves current user info
// @Summary Get current user
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.User
// @Router /api/auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User tidak ditemukan",
		})
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

type ChangePasswordInput struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword handles password change
// @Summary Change password
// @Tags Auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body ChangePasswordInput true "Password change input"
// @Success 200 {object} map[string]string
// @Router /api/auth/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var input ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password lama dan baru wajib diisi (minimal 8 karakter)",
		})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User tidak ditemukan",
		})
		return
	}

	err := h.authService.ChangePassword(c.Request.Context(), userID, input.OldPassword, input.NewPassword)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Password lama tidak sesuai",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengubah password",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password berhasil diubah",
	})
}
