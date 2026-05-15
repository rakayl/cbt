package question_banks

import (
	"io"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h Handler) DownloadTemplate(c *fiber.Ctx) error {
	filename := "template-bank-soal-pilihan-ganda.csv"
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set(fiber.HeaderLastModified, time.Now().UTC().Format(time.RFC1123))
	return c.SendString(QuestionImportTemplateCSV())
}

func (h Handler) ImportQuestions(c *fiber.Ctx) error {
	questionBankID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid question bank id", nil)
	}
	var lecturerID *uuid.UUID
	if raw := c.FormValue("lecturer_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return response.Error(c, fiber.StatusBadRequest, "invalid lecturer id", nil)
		}
		lecturerID = &parsed
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
	payload, err := io.ReadAll(src)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "cannot read file", err)
	}
	out, err := h.service.(*service).ImportQuestions(c.Context(), shared.TenantID(c), questionBankID, shared.UserID(c), shared.Permissions(c), lecturerID, file.Filename, payload)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "question import failed", err)
	}
	return response.Created(c, out)
}

func (h Handler) UploadMedia(c *fiber.Ctx) error {
	questionBankID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid question bank id", nil)
	}
	if err := h.service.(*service).ensureQuestionBankWritable(c.Context(), shared.TenantID(c), questionBankID, shared.UserID(c), shared.Permissions(c)); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "question bank access denied", err)
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
	out, err := h.service.(*service).UploadMedia(c.Context(), shared.TenantID(c), questionBankID, file.Filename, file.Header.Get("Content-Type"), src, file.Size)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "media upload failed", err)
	}
	return response.Created(c, out)
}
