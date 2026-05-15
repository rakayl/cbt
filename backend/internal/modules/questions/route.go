package questions

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
	group := api.Group("/questions", middleware.JWT(deps.Config), ModuleMiddleware())
	group.Get("/ws", websocket.New(WebSocketHandler(deps)))
	group.Get("/", middleware.RequirePermission(PermissionRead), h.List)
	group.Post("/", middleware.RequirePermission(PermissionWrite), h.Create)
	group.Get("/media/:media_id/content", middleware.RequirePermission(PermissionRead), h.MediaContent)
	group.Delete("/media/:media_id", middleware.RequirePermission(PermissionWrite), h.DeleteMedia)
	group.Get("/:id/usage", middleware.RequirePermission(PermissionRead), h.Usage)
	group.Get("/:id/versions", middleware.RequirePermission(PermissionRead), h.Versions)
	group.Get("/:id", middleware.RequirePermission(PermissionRead), h.Get)
	group.Put("/:id", middleware.RequirePermission(PermissionWrite), h.Update)
	group.Delete("/:id", middleware.RequirePermission(PermissionWrite), h.Delete)
	group.Post("/:id/media", middleware.RequirePermission(PermissionWrite), h.UploadMedia)
}
