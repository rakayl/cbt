package reports

import (
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/response"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/gofiber/fiber/v2"
)

func (h Handler) Export(c *fiber.Ctx) error {
	var req ExportReportRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid payload", err)
	}
	file, err := h.service.(*service).Export(c.Context(), shared.TenantID(c), shared.UserID(c), shared.Permissions(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "report export failed", err)
	}
	c.Set(fiber.HeaderContentType, file.ContentType)
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+file.Filename+`"`)
	return c.Send(file.Bytes)
}
