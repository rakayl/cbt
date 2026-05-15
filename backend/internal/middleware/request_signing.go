package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/gofiber/fiber/v2"
)

func RequestSigning(cfg config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !cfg.RequestSigningRequired || c.Method() == fiber.MethodGet {
			return c.Next()
		}
		sig := c.Get("X-Request-Signature")
		ts := c.Get("X-Request-Timestamp")
		if sig == "" || ts == "" {
			return response.Error(c, fiber.StatusUnauthorized, "missing request signature", nil)
		}
		mac := hmac.New(sha256.New, []byte(cfg.RequestSigningSecret))
		mac.Write([]byte(ts + ":" + c.Method() + ":" + c.Path() + ":" + string(c.Body())))
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(sig), []byte(expected)) {
			return response.Error(c, fiber.StatusUnauthorized, "invalid request signature", nil)
		}
		return c.Next()
	}
}
