package middleware

import (
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/security"
	"github.com/gofiber/fiber/v2"
	"strings"
)

func JWT(cfg config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		token := ""
		if strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		}
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			return response.Error(c, fiber.StatusUnauthorized, "missing bearer token", nil)
		}
		claims, err := security.ParseJWT(token, cfg.JWTAccessSecret)
		if err != nil {
			return response.Error(c, fiber.StatusUnauthorized, "invalid token", nil)
		}
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("session_id", claims.SessionID)
		c.Locals("permissions", claims.Permissions)
		return c.Next()
	}
}
