package screen_recordings

import "github.com/gofiber/fiber/v2"

func ModuleMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error { c.Set("X-CBT-Module", "screen_recordings"); return c.Next() }
}
