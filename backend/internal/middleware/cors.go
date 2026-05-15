package middleware

import (
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"strings"
)

func CORS(cfg config.Config) fiber.Handler {
	return cors.New(cors.Config{AllowOrigins: strings.Join(cfg.CORSAllowedOrigins, ","), AllowCredentials: true, AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Tenant-ID, X-Request-Signature, X-Request-Timestamp, X-CSRF-Token", AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS"})
}
