package middleware

import (
	"net/http"
	"strings"

	"cbt-enterprise/pkg/jwt"
	"cbt-enterprise/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func JWTAuth(j *jwt.JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Error(c, http.StatusUnauthorized, "authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, http.StatusUnauthorized, "invalid authorization format")
			c.Abort()
			return
		}

		claims, err := j.Validate(parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "invalid token claims")
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.MustGet("role").(string)
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}
		response.Error(c, http.StatusForbidden, "access denied")
		c.Abort()
	}
}
