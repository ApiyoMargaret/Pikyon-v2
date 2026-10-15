package auth

import (
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"
)

type Handler struct {
    service Service
}

func NewHandler(service Service) *Handler {
    return &Handler{service: service}
}

// RegisterRoutes registers all authentication routes under a single /auth group.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
    auth := router.Group("/auth")
    {
        auth.POST("/register", h.Register)
        auth.POST("/login", h.Login)
        auth.POST("/refresh", h.RefreshToken)
        auth.POST("/logout", authMiddleware, h.Logout) // Protected inline
    }
}

func (h *Handler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload or missing required fields"})
        return
    }

    userResp, err := h.service.Register(c.Request.Context(), req.Email, req.Password, req.Name)
    if err != nil {
        if errors.Is(err, ErrEmailAlreadyExists) {
            c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
        return
    }

    c.JSON(http.StatusCreated, userResp)
}

func (h *Handler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login credentials format"})
        return
    }

    deviceInfo := c.GetHeader("User-Agent")
    clientIP := c.ClientIP()

    tokens, userResp, err := h.service.Login(c.Request.Context(), req.Email, req.Password, deviceInfo, clientIP)
    if err != nil {
        if errors.Is(err, ErrInvalidCredentials) {
            c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate user"})
        return
    }

    resp := LoginResponse{
        AccessToken:  tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
        ExpiresAt:    tokens.ExpiresAt,
        User:         userResp,
    }

    c.JSON(http.StatusOK, resp)
}

func (h *Handler) RefreshToken(c *gin.Context) {
    var req RefreshTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token refresh request"})
        return
    }

    deviceInfo := c.GetHeader("User-Agent")
    clientIP := c.ClientIP()

    tokens, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken, deviceInfo, clientIP)
    if err != nil {
        switch {
        case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidToken):
            c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        case errors.Is(err, ErrSessionBlocked):
            c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
        default:
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token"})
        }
        return
    }

    resp := RefreshResponse{
        AccessToken:  tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
        ExpiresAt:    tokens.ExpiresAt,
    }

    c.JSON(http.StatusOK, resp)
}

func (h *Handler) Logout(c *gin.Context) {
    var req RefreshTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid logout request payload"})
        return
    }

    // Extract authenticated user ID from context to prevent cross-tenant session eviction
    userID, err := GetAuthUserID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized user context"})
        return
    }

    if err := h.service.Logout(c.Request.Context(), userID, req.RefreshToken); err != nil {
        if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrSessionNotFound) {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout session"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "successfully logged out"})
}