package middlewares

import (
	"backend/helpers"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Middleware(jwtKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Request.Header.Get("Authorization")

		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Authorization header required",
			})
			return
		}

		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid authorization format",
			})
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Token is required",
			})
			return
		}

		result, errDecode := helpers.DecodeJWT(tokenString, jwtKey)
		if errDecode != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid authorization format",
			})
			return
		}

		userId, ok := result["user_id"].(string)
		if !ok || userId == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid token claims",
			})
			return
		}

		c.Set("user_id", userId)
		c.Next()
	}
}
