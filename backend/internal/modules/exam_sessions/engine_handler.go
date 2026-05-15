package exam_sessions

import (
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h Handler) StartExam(c *fiber.Ctx) error {
	var req StartExamRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.(*service).StartExam(c.Context(), shared.TenantID(c), shared.UserID(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot start exam", err)
	}
	return response.Created(c, out)
}

func (h Handler) AutosaveAnswer(c *fiber.Ctx) error {
	var req AutosaveAnswerRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.(*service).AutosaveAnswer(c.Context(), shared.TenantID(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "autosave failed", err)
	}
	return response.OK(c, out)
}

func (h Handler) Reconnect(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid session id", nil)
	}
	var req ReconnectRequest
	_ = c.BodyParser(&req)
	req.SessionID = sessionID
	out, err := h.service.(*service).Reconnect(c.Context(), shared.TenantID(c), shared.UserID(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "reconnect failed", err)
	}
	return response.OK(c, out)
}

func (h Handler) SubmitExam(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid session id", nil)
	}
	var req SubmitExamRequest
	_ = c.BodyParser(&req)
	out, err := h.service.(*service).SubmitExam(c.Context(), shared.TenantID(c), shared.UserID(c), sessionID, req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "submit failed", err)
	}
	return response.OK(c, out)
}

func (h Handler) SessionQuestions(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid session id", nil)
	}
	out, err := h.service.SessionQuestions(c.Context(), shared.TenantID(c), shared.UserID(c), sessionID)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load exam questions", err)
	}
	return response.OK(c, out)
}

func (h Handler) AutoSubmitExpired(c *fiber.Ctx) error {
	count, err := h.service.(*service).AutoSubmitExpiredSessions(c.Context(), shared.TenantID(c))
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "auto submit failed", err)
	}
	return response.OK(c, fiber.Map{"submitted": count})
}
