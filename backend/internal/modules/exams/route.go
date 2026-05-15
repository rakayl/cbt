package exams

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
	group := api.Group("/exams", middleware.JWT(deps.Config), ModuleMiddleware())
	group.Get("/ws", websocket.New(WebSocketHandler(deps)))
	group.Get("/", middleware.RequirePermission(PermissionRead), h.List)
	group.Post("/", middleware.RequirePermission(PermissionWrite), h.Create)
	group.Post("/join-by-code", middleware.RequirePermission("exams:join"), h.JoinByCode)
	group.Get("/student", middleware.RequirePermission("exams:join"), h.StudentExams)
	group.Get("/:id/invites", middleware.RequirePermission("exams:invite"), h.ListInvites)
	group.Post("/:id/invite-students", middleware.RequirePermission("exams:invite"), h.InviteStudents)
	group.Post("/:id/share-code", middleware.RequirePermission("exams:invite"), h.ShareCode)
	group.Post("/:id/publish", middleware.RequirePermission(PermissionWrite), h.Publish)
	group.Get("/:id/revisions", middleware.RequirePermission(PermissionRead), h.ListRevisions)
	group.Post("/:id/revisions", middleware.RequirePermission(PermissionWrite), h.CreateRevision)
	group.Get("/:id", middleware.RequirePermission(PermissionRead), h.Get)
	group.Put("/:id", middleware.RequirePermission(PermissionWrite), h.Update)
	group.Delete("/:id", middleware.RequirePermission(PermissionWrite), h.Delete)
}
