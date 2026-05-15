package enrollment

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
	filters := EnrollmentFilters{
		StudentID:   parseUUID(c.Query("student_id")),
		ClassRoomID: parseUUID(c.Query("class_room_id")),
	}
	out, err := h.service.List(c.Context(), shared.TenantID(c), q, filters)
	if err != nil {
		return err
	}
	return response.OK(c, out)
}

func (h Handler) StudentHistory(c *fiber.Ctx) error {
	studentID, err := uuid.Parse(c.Params("student_id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid student id", nil)
	}
	q := pagination.New(c.Query("page"), c.Query("limit"), c.Query("search"), c.Query("sort"))
	out, err := h.service.StudentHistory(c.Context(), shared.TenantID(c), studentID, q)
	if err != nil {
		return err
	}
	return response.OK(c, out)
}

func (h Handler) MyClasses(c *fiber.Ctx) error {
	q := pagination.New(c.Query("page"), c.Query("limit"), c.Query("search"), c.Query("sort"))
	out, err := h.service.MyClasses(c.Context(), shared.TenantID(c), shared.UserID(c), q)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "cannot load student classes", err)
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
	var req CreateEnrollmentRequest
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
	var req UpdateEnrollmentRequest
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

func parseUUID(value string) uuid.UUID {
	id, _ := uuid.Parse(value)
	return id
}
