package billing

import (
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
)

func (h Handler) Usage(c *fiber.Ctx) error {
	out, err := h.service.(*service).Usage(c.Context(), shared.TenantID(c))
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "usage lookup failed", err)
	}
	return response.OK(c, out)
}

func (h Handler) CheckoutIntent(c *fiber.Ctx) error {
	var req CheckoutIntentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.(*service).CreateCheckoutIntent(c.Context(), shared.TenantID(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "checkout intent failed", err)
	}
	return response.Created(c, out)
}

func (h Handler) ChangePlan(c *fiber.Ctx) error {
	var req ChangePlanRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.(*service).ChangePlan(c.Context(), shared.TenantID(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "plan update failed", err)
	}
	return response.OK(c, out)
}
