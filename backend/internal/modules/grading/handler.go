package grading

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
	out, err := h.service.List(c.Context(), shared.TenantID(c), q)
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
	out, err := h.service.Get(c.Context(), shared.TenantID(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, out)
}
func (h Handler) Create(c *fiber.Ctx) error {
	var req CreateGradingRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.Create(c.Context(), shared.TenantID(c), req)
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
	var req UpdateGradingRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.Update(c.Context(), shared.TenantID(c), id, req)
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
	if err := h.service.Delete(c.Context(), shared.TenantID(c), id); err != nil {
		return err
	}
	return response.OK(c, fiber.Map{"deleted": true})
}

func (h Handler) SessionGrades(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("session_id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid session id", nil)
	}
	out, err := h.service.SessionGrades(c.Context(), shared.TenantID(c), sessionID)
	if err != nil {
		return err
	}
	return response.OK(c, out)
}

func (h Handler) ReviewSessions(c *fiber.Ctx) error {
	q := pagination.New(c.Query("page"), c.Query("limit"), c.Query("search"), c.Query("sort"))
	var examID uuid.UUID
	if raw := c.Query("exam_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
		}
		examID = parsed
	}
	out, err := h.service.ReviewSessions(c.Context(), shared.TenantID(c), shared.UserID(c), shared.Permissions(c), q, examID)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load review sessions", err)
	}
	return response.OK(c, out)
}

func (h Handler) ReviewSessionDetail(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("session_id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid session id", nil)
	}
	out, err := h.service.ReviewSessionDetail(c.Context(), shared.TenantID(c), shared.UserID(c), shared.Permissions(c), sessionID)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load review detail", err)
	}
	return response.OK(c, out)
}

func (h Handler) ReleaseSessionResult(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("session_id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid session id", nil)
	}
	out, err := h.service.ReleaseSessionResult(c.Context(), shared.TenantID(c), shared.UserID(c), shared.Permissions(c), sessionID)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "release result failed", err)
	}
	return response.OK(c, out)
}

func (h Handler) ManualScore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid id", nil)
	}
	var req ManualScoreRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.ManualScore(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "manual grading failed", err)
	}
	return response.OK(c, out)
}
