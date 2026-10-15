package auth

import (
    "errors"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

const (
    AuthorizationHeader = "Authorization"
    AuthUserIDKey       = "user_id"
)

var (
    ErrMissingAuthHeader = errors.New("authorization header is missing")
    ErrInvalidAuthFormat = errors.New("authorization header format must be Bearer <token>")
    ErrUnauthorized      = errors.New("unauthorized user context")
)

// AuthMiddleware creates a Gin middleware function to authenticate requests via JWT access tokens.
func AuthMiddleware(service Service) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader(AuthorizationHeader)
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": ErrMissingAuthHeader.Error()})
            c.Abort()
            return
        }

        fields := strings.Fields(authHeader)
        if len(fields) < 2 || strings.ToLower(fields[0]) != "bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": ErrInvalidAuthFormat.Error()})
            c.Abort()
            return
        }

        accessToken := fields[1]
        claims, err := service.ValidateAccessToken(accessToken)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired access token"})
            c.Abort()
            return
        }

        // Store user ID in context for downstream handlers
        c.Set(AuthUserIDKey, claims.UserID)
        c.Next()
    }
}

// GetAuthUserID safely retrieves the authenticated user ID from the Gin request context.
func GetAuthUserID(c *gin.Context) (string, error) {
    userIDVal, exists := c.Get(AuthUserIDKey)
    if !exists {
        return "", ErrUnauthorized
    }

    userID, ok := userIDVal.(string)
    if !ok || userID == "" {
        return "", ErrUnauthorized
    }

    return userID, nil
}