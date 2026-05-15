package question_banks

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
	group := api.Group("/question-banks", middleware.JWT(deps.Config), ModuleMiddleware())
	group.Get("/ws", websocket.New(WebSocketHandler(deps)))
	group.Get("/template", middleware.RequirePermission(PermissionRead), h.DownloadTemplate)
	group.Post("/:id/import", middleware.RequirePermission(PermissionWrite), h.ImportQuestions)
	group.Post("/:id/media", middleware.RequirePermission(PermissionWrite), h.UploadMedia)
	group.Get("/", middleware.RequirePermission(PermissionRead), h.List)
	group.Post("/", middleware.RequirePermission(PermissionWrite), h.Create)
	group.Get("/:id", middleware.RequirePermission(PermissionRead), h.Get)
	group.Put("/:id", middleware.RequirePermission(PermissionWrite), h.Update)
	group.Delete("/:id", middleware.RequirePermission(PermissionWrite), h.Delete)
}
