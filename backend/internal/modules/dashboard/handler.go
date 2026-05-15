package dashboard

import (
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) Summary(c *fiber.Ctx) error {
	out, err := h.service.Summary(c.Context(), shared.TenantID(c), shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load dashboard summary", err)
	}
	return response.OK(c, out)
}
