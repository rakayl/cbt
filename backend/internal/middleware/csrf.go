package middleware

import (
	"strings"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/gofiber/fiber/v2"
)

func CSRF() fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}
		if isPublicAuthPath(c.Path()) {
			return c.Next()
		}
		if c.Get("Authorization") != "" {
			return c.Next()
		}
		cookie := c.Cookies("csrf_token")
		header := c.Get("X-CSRF-Token")
		if cookie == "" || header == "" || cookie != header {
			return response.Error(c, fiber.StatusForbidden, "csrf token invalid", nil)
		}
		return c.Next()
	}
}

func isPublicAuthPath(path string) bool {
	path = strings.TrimSuffix(path, "/")
	switch path {
	case "/api/v1/auth/login", "/api/v1/auth/register", "/api/v1/auth/forgot-password", "/api/v1/auth/refresh":
		return true
	default:
		return false
	}
}
