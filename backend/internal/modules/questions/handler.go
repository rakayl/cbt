package questions

import (
	"io"

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
func (h Handler) Usage(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid id", nil)
	}
	out, err := h.service.Usage(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return err
	}
	return response.OK(c, out)
}
func (h Handler) Versions(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid id", nil)
	}
	out, err := h.service.Versions(c.Context(), shared.TenantID(c), id, shared.UserID(c), shared.Permissions(c))
	if err != nil {
		return err
	}
	return response.OK(c, out)
}
func (h Handler) Create(c *fiber.Ctx) error {
	var req CreateQuestionRequest
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
	var req UpdateQuestionRequest
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

func (h Handler) UploadMedia(c *fiber.Ctx) error {
	questionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid question id", nil)
	}
	var optionID *uuid.UUID
	if raw := c.FormValue("option_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return response.Error(c, fiber.StatusBadRequest, "invalid option id", nil)
		}
		optionID = &parsed
	}
	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "file is required", err)
	}
	src, err := file.Open()
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "cannot open file", err)
	}
	defer src.Close()
	out, err := h.service.UploadMedia(
		c.Context(),
		shared.TenantID(c),
		shared.UserID(c),
		shared.Permissions(c),
		questionID,
		optionID,
		c.FormValue("usage_type", "question"),
		file.Filename,
		file.Header.Get("Content-Type"),
		src,
		file.Size,
	)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "upload failed", err)
	}
	return response.Created(c, out)
}

func (h Handler) DeleteMedia(c *fiber.Ctx) error {
	mediaID, err := uuid.Parse(c.Params("media_id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid media id", nil)
	}
	if err := h.service.DeleteMedia(c.Context(), shared.TenantID(c), shared.UserID(c), mediaID, shared.Permissions(c)); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "delete media failed", err)
	}
	return response.OK(c, fiber.Map{"deleted": true})
}

func (h Handler) MediaContent(c *fiber.Ctx) error {
	mediaID, err := uuid.Parse(c.Params("media_id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid media id", nil)
	}
	body, contentType, err := h.service.MediaContent(c.Context(), shared.TenantID(c), mediaID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "media not found", err)
	}
	defer body.Close()
	c.Set(fiber.HeaderContentType, contentType)
	_, err = io.Copy(c.Response().BodyWriter(), body)
	return err
}
