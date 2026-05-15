package billing

import (
	"github.com/cbt-ai/enterprise-cbt/internal/middleware"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(api fiber.Router, deps shared.Deps) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo, deps)
	h := NewHandler(svc)
	group := api.Group("/billing", middleware.JWT(deps.Config), ModuleMiddleware())
	group.Get("/ws", websocket.New(WebSocketHandler(deps)))
	group.Get("/usage", middleware.RequirePermission(PermissionRead), h.Usage)
	group.Post("/checkout-intent", middleware.RequirePermission(PermissionWrite), h.CheckoutIntent)
	group.Post("/change-plan", middleware.RequirePermission(PermissionWrite), h.ChangePlan)
	group.Get("/", middleware.RequirePermission(PermissionRead), h.List)
	group.Post("/", middleware.RequirePermission(PermissionWrite), h.Create)
	group.Get("/:id", middleware.RequirePermission(PermissionRead), h.Get)
	group.Put("/:id", middleware.RequirePermission(PermissionWrite), h.Update)
	group.Delete("/:id", middleware.RequirePermission(PermissionWrite), h.Delete)
}
