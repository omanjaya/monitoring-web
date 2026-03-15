package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
)

type KeywordHandler struct {
	keywordRepo *mysql.KeywordRepository
}

func NewKeywordHandler(keywordRepo *mysql.KeywordRepository) *KeywordHandler {
	return &KeywordHandler{
		keywordRepo: keywordRepo,
	}
}

// ListKeywords retrieves all keywords
// @Summary List keywords
// @Tags Keywords
// @Security BearerAuth
// @Produce json
// @Param category query string false "Filter by category"
// @Success 200 {array} domain.Keyword
// @Router /api/keywords [get]
func (h *KeywordHandler) ListKeywords(c *gin.Context) {
	category := c.Query("category")

	var keywords []domain.Keyword
	var err error

	if category != "" {
		keywords, err = h.keywordRepo.GetByCategory(c.Request.Context(), category)
	} else {
		keywords, err = h.keywordRepo.GetAll(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data keyword",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": keywords})
}

// CreateKeyword creates a new keyword
// @Summary Create keyword
// @Tags Keywords
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.Keyword true "Keyword data"
// @Success 201 {object} domain.Keyword
// @Router /api/keywords [post]
func (h *KeywordHandler) CreateKeyword(c *gin.Context) {
	var input domain.Keyword
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	// Set default values
	if input.Weight == 0 {
		input.Weight = 5
	}
	input.IsActive = true

	id, err := h.keywordRepo.Create(c.Request.Context(), &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat keyword: " + err.Error(),
		})
		return
	}

	input.ID = id
	c.JSON(http.StatusCreated, input)
}

// DeleteKeyword deletes a keyword
// @Summary Delete keyword
// @Tags Keywords
// @Security BearerAuth
// @Param id path int true "Keyword ID"
// @Success 200 {object} map[string]string
// @Router /api/keywords/{id} [delete]
func (h *KeywordHandler) DeleteKeyword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	if err := h.keywordRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menghapus keyword",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Keyword berhasil dihapus",
	})
}

// BulkImportKeywords imports multiple keywords at once
// @Summary Bulk import keywords
// @Tags Keywords
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body []domain.Keyword true "List of keywords"
// @Success 200 {object} map[string]interface{}
// @Router /api/keywords/bulk [post]
func (h *KeywordHandler) BulkImportKeywords(c *gin.Context) {
	var input []domain.Keyword
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	if len(input) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data kosong",
		})
		return
	}

	type BulkError struct {
		Keyword string `json:"keyword"`
		Error   string `json:"error"`
	}

	var created []string
	var skipped []string
	var failed []BulkError

	// Get existing keywords to check duplicates
	existing, _ := h.keywordRepo.GetAll(c.Request.Context())
	existingMap := make(map[string]bool)
	for _, kw := range existing {
		existingMap[kw.Keyword] = true
	}

	for _, kw := range input {
		if kw.Keyword == "" {
			failed = append(failed, BulkError{Keyword: kw.Keyword, Error: "keyword kosong"})
			continue
		}

		if existingMap[kw.Keyword] {
			skipped = append(skipped, kw.Keyword)
			continue
		}

		// Set defaults
		if kw.Weight == 0 {
			kw.Weight = 5
		}
		if kw.Category == "" {
			kw.Category = "custom"
		}
		kw.IsActive = true

		_, err := h.keywordRepo.Create(c.Request.Context(), &kw)
		if err != nil {
			failed = append(failed, BulkError{Keyword: kw.Keyword, Error: err.Error()})
			continue
		}

		created = append(created, kw.Keyword)
		existingMap[kw.Keyword] = true
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"created": created,
			"skipped": skipped,
			"failed":  failed,
		},
	})
}
