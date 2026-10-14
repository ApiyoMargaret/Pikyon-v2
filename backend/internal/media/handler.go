package media

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all media domain endpoints onto the Gin router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	mediaGroup := rg.Group("/media")
	{
		mediaGroup.POST("/presign", h.GeneratePresignedURL)
		mediaGroup.POST("/complete", h.CompleteUpload)
	}
}

// Helper to extract authenticated user ID from Gin context
func getUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}
	idStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user session"})
		return "", false
	}
	return idStr, true
}

func (h *Handler) GeneratePresignedURL(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req PresignedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.GeneratePresignedURL(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *Handler) CompleteUpload(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.CompleteUpload(c.Request.Context(), userID, &req)
	if err != nil {
		if err.Error() == "unauthorized access to pending upload session" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "pending upload not found, expired, or already completed" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}