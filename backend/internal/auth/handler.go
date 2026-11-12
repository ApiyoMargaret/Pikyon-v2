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

// RegisterRoutes registers public and protected authentication routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	auth := router.Group("/auth")
	{
		// Google OAuth
		auth.GET("/google/start", h.GoogleOAuthStart)
		auth.GET("/google/callback", h.GoogleOAuthCallback)

		// Registration and login
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)

		// Token refresh
		auth.POST("/refresh", h.RefreshToken)
	}

	// Protect logout endpoint to ensure only authenticated users can delete sessions
	protectedAuth := router.Group("/auth")
	protectedAuth.Use(authMiddleware)
	{
		protectedAuth.POST("/logout", h.Logout)
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload or missing required fields"})
		return
	}

	user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login credentials format"})
		return
	}

	userAgent := c.GetHeader("User-Agent")
	// Note: Engine.SetTrustedProxies() must be configured at router setup to prevent header spoofing
	clientIP := c.ClientIP()

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, userAgent, clientIP)
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
		User:         user,
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token refresh request"})
		return
	}

	userAgent := c.GetHeader("User-Agent")
	// Note: Engine.SetTrustedProxies() must be configured at router setup to prevent header spoofing
	clientIP := c.ClientIP()

	tokens, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken, userAgent, clientIP)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrInvalidCredentials):
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

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully logged out"})
}

// PinSetRequest is the payload for setting a memory PIN.
type PinSetRequest struct {
	MemoryID  string `json:"memory_id" binding:"required,uuid"`
	PinCode   string `json:"pin_code" binding:"required,min=4,max=4"`
}

// PinVerifyRequest is the payload for verifying a memory PIN.
type PinVerifyRequest struct {
	MemoryID  string `json:"memory_id" binding:"required,uuid"`
	PinCode   string `json:"pin_code" binding:"required,min=4,max=4"`
}

// PinVerifyResponse is the response for PIN verification.
type PinVerifyResponse struct {
	Success bool `json:"success"`
	Verified bool `json:"verified"`
}

// PinVerify verifies a 4-digit PIN for a memory.
func (h *Handler) PinVerify(c *gin.Context) {
	var req PinVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid PIN verification request"})
		return
	}

	verified, err := h.service.PinVerify(c.Request.Context(), req.MemoryID, req.PinCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify PIN: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, PinVerifyResponse{Success: true, Verified: verified})
}

// GoogleOAuthCallback handles the Google OAuth2 callback exchange.
func (h *Handler) GoogleOAuthCallback(c *gin.Context) {
	exchangeCode := c.Query("code")
	if exchangeCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization code"})
		return
	}

	tokens, user, err := h.service.GoogleOAuthCallback(c.Request.Context(), exchangeCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google OAuth callback failed: " + err.Error()})
		return
	}

	resp := LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt,
		User:         user,
	}

	c.JSON(http.StatusOK, resp)
}