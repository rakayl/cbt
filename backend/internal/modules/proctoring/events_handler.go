package proctoring

import (
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h Handler) IngestEvent(c *fiber.Ctx) error {
	var req ProctoringEventRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.(*service).IngestEvent(c.Context(), shared.TenantID(c), shared.UserID(c), shared.Permissions(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "proctoring event rejected", err)
	}
	return response.Created(c, out)
}

func (h Handler) UploadSnapshot(c *fiber.Ctx) error {
	var req ProctoringSnapshotRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.(*service).UploadSnapshot(c.Context(), shared.TenantID(c), shared.UserID(c), shared.Permissions(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "snapshot rejected", err)
	}
	return response.Created(c, out)
}

func (h Handler) ListEvents(c *fiber.Ctx) error {
	q := pagination.New(c.Query("page"), c.Query("limit"), c.Query("search"), c.Query("sort"))
	out, err := h.service.(*service).ListEvents(c.Context(), shared.TenantID(c), q, c.Query("event_type"), c.Query("severity"), c.Query("session_id"))
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot list proctoring events", err)
	}
	return response.OK(c, out)
}

func (h Handler) SessionTimeline(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("session_id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid session id", nil)
	}
	out, err := h.service.SessionTimeline(c.Context(), shared.TenantID(c), sessionID)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load proctoring timeline", err)
	}
	return response.OK(c, out)
}
