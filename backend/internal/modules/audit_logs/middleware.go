package audit_logs

import "github.com/gofiber/fiber/v2"

func ModuleMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error { c.Set("X-CBT-Module", "audit_logs"); return c.Next() }
}
