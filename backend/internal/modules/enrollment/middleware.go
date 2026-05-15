package enrollment

import "github.com/gofiber/fiber/v2"

func ModuleMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error { c.Set("X-CBT-Module", "enrollment"); return c.Next() }
}
