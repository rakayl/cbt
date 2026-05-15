package auth

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
	public := api.Group("/auth")
	public.Post("/login", h.Login)
	public.Post("/register", h.Register)
	public.Post("/forgot-password", h.ForgotPassword)
	public.Post("/refresh", h.Refresh)
	private := api.Group("/auth", middleware.JWT(deps.Config))
	private.Post("/logout", h.Logout)
	private.Get("/sessions", middleware.RequirePermission(PermissionRead), h.List)
	private.Get("/ws", websocket.New(WebSocketHandler(deps)))
}
