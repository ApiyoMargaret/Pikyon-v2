package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeaderKey  = "authorization"
	AuthorizationTypeBearer = "bearer"
	AuthorizationPayloadKey = "authorization_payload"
	UserIDKey               = "user_id"
)

// AuthMiddleware creates a Gin middleware for JWT access token authentication.
func AuthMiddleware(service Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if len(authHeader) == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			return
		}

		fields := strings.Fields(authHeader)
		if len(fields) != 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != AuthorizationTypeBearer {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unsupported authorization type"})
			return
		}

		accessToken := fields[1]
		claims, err := service.ValidateAccessToken(accessToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired access token"})
			return
		}

		// Inject verified payload and user ID into Gin context for downstream handlers
		c.Set(AuthorizationPayloadKey, claims)
		c.Set(UserIDKey, claims.UserID)

		c.Next()
	}
}

// GetAuthUserID extracts the authenticated user ID from the Gin context.
func GetAuthUserID(c *gin.Context) (string, error) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return "", errors.New("user ID not found in context")
	}

	idStr, ok := userID.(string)
	if !ok {
		return "", errors.New("invalid user ID type in context")
	}

	return idStr, nil
}