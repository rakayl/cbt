package grading

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
	group := api.Group("/grading", middleware.JWT(deps.Config), ModuleMiddleware())
	group.Get("/ws", websocket.New(WebSocketHandler(deps)))
	group.Get("/", middleware.RequirePermission(PermissionRead), h.List)
	group.Post("/", middleware.RequirePermission(PermissionWrite), h.Create)
	group.Get("/review/sessions", middleware.RequirePermission(PermissionRead), h.ReviewSessions)
	group.Get("/review/sessions/:session_id", middleware.RequirePermission(PermissionRead), h.ReviewSessionDetail)
	group.Put("/review/sessions/:session_id/release", middleware.RequirePermission(PermissionWrite), h.ReleaseSessionResult)
	group.Get("/sessions/:session_id", middleware.RequirePermission(PermissionRead), h.SessionGrades)
	group.Get("/:id", middleware.RequirePermission(PermissionRead), h.Get)
	group.Put("/:id/manual-score", middleware.RequirePermission(PermissionWrite), h.ManualScore)
	group.Put("/:id", middleware.RequirePermission(PermissionWrite), h.Update)
	group.Delete("/:id", middleware.RequirePermission(PermissionWrite), h.Delete)
}
