package middleware

import (
	"github.com/gin-gonic/gin"
)

// Recovery returns a middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		c.JSON(500, gin.H{
			"error":   "Internal server error",
			"message": "An unexpected error occurred",
		})
		c.Abort()
	})
}
