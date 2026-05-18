package exams

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

func (h Handler) StudentExams(c *fiber.Ctx) error {
	q := pagination.New(c.Query("page"), c.Query("limit"), c.Query("search"), c.Query("sort"))
	out, err := h.service.StudentExams(c.Context(), shared.TenantID(c), shared.UserID(c), q)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load student exams", err)
	}
	return response.OK(c, out)
}

func (h Handler) Rankings(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
	}
	var classID uuid.UUID
	if rawClassID := c.Query("class_id"); rawClassID != "" {
		classID, err = uuid.Parse(rawClassID)
		if err != nil {
			return response.Error(c, fiber.StatusBadRequest, "invalid class id", nil)
		}
	}
	q := pagination.New(c.Query("page"), c.Query("limit"), c.Query("search"), c.Query("sort"))
	out, err := h.service.Rankings(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c), q, classID)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load exam rankings", err)
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
	var req CreateExamRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.Create(c.Context(), shared.TenantID(c), shared.UserID(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "validation or persistence failed", err)
	}
	return response.Created(c, out)
}

func (h Handler) InviteStudents(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
	}
	var req InviteStudentsRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.InviteStudents(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "invite failed", err)
	}
	return response.OK(c, out)
}

func (h Handler) ListInvites(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
	}
	out, err := h.service.ListInvites(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load invites", err)
	}
	return response.OK(c, out)
}

func (h Handler) InviteRoster(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
	}
	out, err := h.service.InviteRoster(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load invite roster", err)
	}
	return response.OK(c, out)
}

func (h Handler) UpdateAccessStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
	}
	var req UpdateExamAccessRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.UpdateAccessStatus(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot update exam access status", err)
	}
	return response.OK(c, out)
}

func (h Handler) ShareCode(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
	}
	out, err := h.service.ShareCode(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "share code failed", err)
	}
	return response.OK(c, out)
}

func (h Handler) Publish(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
	}
	var req PublishExamRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
		}
	}
	out, err := h.service.Publish(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "publish failed", err)
	}
	return response.OK(c, out)
}

func (h Handler) CreateRevision(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
	}
	out, err := h.service.CreateRevision(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "revision failed", err)
	}
	return response.Created(c, out)
}

func (h Handler) ListRevisions(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid exam id", nil)
	}
	out, err := h.service.ListRevisions(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load revisions", err)
	}
	return response.OK(c, out)
}

func (h Handler) JoinByCode(c *fiber.Ctx) error {
	var req JoinByCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	out, err := h.service.JoinByCode(c.Context(), shared.TenantID(c), shared.UserID(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "join failed", err)
	}
	return response.OK(c, out)
}
func (h Handler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid id", nil)
	}
	var req UpdateExamRequest
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
		return response.Error(c, fiber.StatusUnprocessableEntity, "delete failed", err)
	}
	return response.OK(c, fiber.Map{"deleted": true})
}
