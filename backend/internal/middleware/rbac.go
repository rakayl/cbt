package middleware

import (
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
)

func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		for _, p := range shared.Permissions(c) {
			if p == permission || p == "*" {
				return c.Next()
			}
		}
		return response.Error(c, fiber.StatusForbidden, "permission denied", map[string]string{"permission": permission})
	}
}
