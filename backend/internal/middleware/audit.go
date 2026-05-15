package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func Audit(log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		log.Info("request", zap.String("method", c.Method()), zap.String("path", c.Path()), zap.Int("status", c.Response().StatusCode()), zap.String("ip", c.IP()))
		return err
	}
}
