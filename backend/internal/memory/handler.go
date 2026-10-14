package memory

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all memory domain endpoints onto the Gin router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	memories := rg.Group("/memories")
	{
		memories.POST("", h.Create)
		memories.GET("", h.List)
		memories.GET("/:id", h.GetByID)
		memories.PATCH("/:id", h.Update)
		memories.DELETE("/:id", h.SoftDelete)
		memories.POST("/:id/restore", h.Restore)
		memories.POST("/bulk-trash", h.BulkTrash)
		memories.DELETE("/trash/empty", h.EmptyTrash)
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

func (h *Handler) Create(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req CreateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.CreateMemory(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	status := c.DefaultQuery("status", "active")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	res, err := h.service.ListMemories(c.Request.Context(), userID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetByID(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	memoryID := c.Param("id")
	res, err := h.service.GetMemoryByID(c.Request.Context(), userID, memoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	memoryID := c.Param("id")
	var req UpdateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.UpdateMemory(c.Request.Context(), userID, memoryID, &req)
	if err != nil {
		if err.Error() == "memory not found or inactive" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) SoftDelete(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	memoryID := c.Param("id")
	if err := h.service.SoftDeleteMemory(c.Request.Context(), userID, memoryID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "memory moved to trash successfully"})
}

func (h *Handler) Restore(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	memoryID := c.Param("id")
	if err := h.service.RestoreMemory(c.Request.Context(), userID, memoryID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "memory restored successfully"})
}

func (h *Handler) BulkTrash(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req BulkActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.BulkTrashMemories(c.Request.Context(), userID, req.MemoryIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "memories moved to trash successfully"})
}

func (h *Handler) EmptyTrash(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	if err := h.service.EmptyTrashBin(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "trash emptied successfully"})
}
