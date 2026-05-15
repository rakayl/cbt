package dashboard

import (
	"github.com/cbt-ai/enterprise-cbt/internal/middleware"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(api fiber.Router, deps shared.Deps) {
	svc := NewService(deps)
	h := NewHandler(svc)
	group := api.Group("/dashboard", middleware.JWT(deps.Config), ModuleMiddleware())
	group.Get("/summary", middleware.RequirePermission(PermissionRead), h.Summary)
}
