package shared

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TenantID(c *fiber.Ctx) uuid.UUID {
	if v, ok := c.Locals("tenant_id").(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}
func UserID(c *fiber.Ctx) uuid.UUID {
	if v, ok := c.Locals("user_id").(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}
func SessionID(c *fiber.Ctx) uuid.UUID {
	if v, ok := c.Locals("session_id").(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}
func Permissions(c *fiber.Ctx) []string {
	if v, ok := c.Locals("permissions").([]string); ok {
		return v
	}
	return nil
}
