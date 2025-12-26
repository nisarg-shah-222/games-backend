package middleware

import (
	"net/http"
	"strings"

	"github.com/games-app/backend/internal/handler"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates a middleware that verifies JWT tokens
func AuthMiddleware(authHandler *handler.AuthHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		userID, email, err := authHandler.VerifyJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("user_id", userID)
		c.Set("email", email)

		c.Next()
	}
}
