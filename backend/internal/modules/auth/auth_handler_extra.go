package auth

import (
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
)

func (h Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	svc := h.service.(*service)
	out, err := svc.Login(c.Context(), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "login failed", err)
	}
	return response.OK(c, out)
}

func (h Handler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	svc := h.service.(*service)
	out, err := svc.Register(c.Context(), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "registration failed", err)
	}
	return response.Created(c, out)
}

func (h Handler) Refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	svc := h.service.(*service)
	out, err := svc.Refresh(c.Context(), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "refresh failed", err)
	}
	return response.OK(c, out)
}
func (h Handler) Logout(c *fiber.Ctx) error {
	if err := h.service.(*service).Logout(c.Context(), shared.SessionID(c)); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "logout failed", err)
	}
	return response.OK(c, fiber.Map{"logged_out": true})
}
func (h Handler) ForgotPassword(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	if err := h.service.(*service).ForgotPassword(c.Context(), req); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "forgot password failed", err)
	}
	return response.OK(c, fiber.Map{"status": "reset_link_queued"})
}
