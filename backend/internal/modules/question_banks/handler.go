package question_banks

import (
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service Service }

func NewHandler(service Service) Handler { return Handler{service: service} }
func (h Handler) List(c *fiber.Ctx) error {
	q := pagination.New(c.Query("page"), c.Query("limit"), c.Query("search"), c.Query("sort"))
	out, err := h.service.List(c.Context(), shared.TenantID(c), q, shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return err
	}
	return response.OK(c, out)
}
func (h Handler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid id", nil)
	}
	out, err := h.service.Get(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return err
	}
	return response.OK(c, out)
}
func (h Handler) Create(c *fiber.Ctx) error {
	var req CreateQuestionBankRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.Create(c.Context(), shared.TenantID(c), shared.UserID(c), shared.Permissions(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation or persistence failed", err)
	}
	return response.Created(c, out)
}
func (h Handler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid id", nil)
	}
	var req UpdateQuestionBankRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.Update(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation or persistence failed", err)
	}
	return response.OK(c, out)
}
func (h Handler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid id", nil)
	}
	if err := h.service.Delete(c.Context(), shared.TenantID(c), shared.UserID(c), id, shared.Permissions(c)); err != nil {
		return err
	}
	return response.OK(c, fiber.Map{"deleted": true})
}
