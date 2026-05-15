package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func Tenant() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Locals("tenant_id") == nil {
			if raw := c.Get("X-Tenant-ID"); raw != "" {
				if id, err := uuid.Parse(raw); err == nil {
					c.Locals("tenant_id", id)
				}
			}
		}
		return c.Next()
	}
}
