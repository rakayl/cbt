package response

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
)

type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

func OK(c *fiber.Ctx, data any) error { return c.JSON(Envelope{Success: true, Data: data}) }
func Created(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(Envelope{Success: true, Data: data})
}
func Error(c *fiber.Ctx, status int, msg string, errors any) error {
	msg, errors = normalizeError(msg, errors)
	return c.Status(status).JSON(Envelope{Success: false, Message: msg, Errors: errors})
}

func normalizeError(message string, input any) (string, any) {
	if input == nil {
		return message, nil
	}
	if normalized, ok := normalizeValidationError(input); ok {
		return "validation failed", normalized
	}
	if normalized, ok := normalizePostgresError(input); ok {
		return normalized["message"].(string), normalized
	}
	if text := strings.TrimSpace(fmt.Sprint(input)); text != "" {
		if normalized, ok := normalizePostgresText(text); ok {
			return normalized["message"].(string), normalized
		}
		if normalized, ok := normalizeValidatorText(text); ok {
			return "validation failed", normalized
		}
		return message, map[string]any{
			"type":    "application_error",
			"message": text,
		}
	}
	return message, input
}

func normalizeValidationError(input any) (map[string]any, bool) {
	var validationErrors validator.ValidationErrors
	if err, ok := input.(error); ok && errors.As(err, &validationErrors) {
		fields := map[string]string{}
		for _, fieldError := range validationErrors {
			field := jsonFieldName(fieldError.Field())
			fields[field] = validationMessage(field, fieldError.Tag(), fieldError.Param())
		}
		return map[string]any{
			"type":    "validation_error",
			"message": "Periksa kembali input form.",
			"fields":  fields,
		}, true
	}
	return nil, false
}

func normalizePostgresError(input any) (map[string]any, bool) {
	var pgErr *pgconn.PgError
	if err, ok := input.(error); ok && errors.As(err, &pgErr) {
		return postgresPayload(pgErr.Code, pgErr.ConstraintName), true
	}
	return nil, false
}

var constraintRegexp = regexp.MustCompile(`constraint "([^"]+)"`)

func normalizePostgresText(text string) (map[string]any, bool) {
	if !strings.Contains(text, "SQLSTATE") && !strings.Contains(strings.ToLower(text), "duplicate key") {
		return nil, false
	}
	code := ""
	for _, candidate := range []string{"23505", "23503", "23514", "23502"} {
		if strings.Contains(text, candidate) {
			code = candidate
			break
		}
	}
	match := constraintRegexp.FindStringSubmatch(text)
	constraint := ""
	if len(match) > 1 {
		constraint = match[1]
	}
	return postgresPayload(code, constraint), true
}

func postgresPayload(code, constraint string) map[string]any {
	field := fieldFromConstraint(constraint)
	switch code {
	case "23505":
		msg := "Data sudah digunakan."
		if field != "" {
			msg = labelForField(field) + " sudah digunakan. Gunakan nilai lain."
		}
		fields := map[string]string{}
		if field != "" {
			fields[field] = msg
		}
		return map[string]any{
			"type":    "unique_violation",
			"message": msg,
			"field":   field,
			"fields":  fields,
		}
	case "23503":
		return map[string]any{
			"type":    "foreign_key_violation",
			"message": "Data referensi tidak valid atau sudah tidak tersedia.",
		}
	case "23502":
		return map[string]any{
			"type":    "not_null_violation",
			"message": "Ada field wajib yang belum diisi.",
		}
	case "23514":
		return map[string]any{
			"type":    "check_violation",
			"message": "Nilai input tidak sesuai aturan database.",
		}
	default:
		return map[string]any{
			"type":    "database_error",
			"message": "Database menolak data yang dikirim.",
		}
	}
}

func normalizeValidatorText(text string) (map[string]any, bool) {
	if !strings.Contains(text, "Error:Field validation") && !strings.Contains(text, "failed on the") {
		return nil, false
	}
	fields := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "Field validation") {
			continue
		}
		field := ""
		if idx := strings.Index(line, "Field validation for '"); idx >= 0 {
			remaining := line[idx+len("Field validation for '"):]
			if end := strings.Index(remaining, "'"); end >= 0 {
				field = jsonFieldName(remaining[:end])
			}
		}
		tag := ""
		if idx := strings.Index(line, "failed on the '"); idx >= 0 {
			remaining := line[idx+len("failed on the '"):]
			if end := strings.Index(remaining, "'"); end >= 0 {
				tag = remaining[:end]
			}
		}
		if field != "" {
			fields[field] = validationMessage(field, tag, "")
		}
	}
	if len(fields) == 0 {
		return nil, false
	}
	return map[string]any{
		"type":    "validation_error",
		"message": "Periksa kembali input form.",
		"fields":  fields,
	}, true
}

func fieldFromConstraint(constraint string) string {
	constraint = strings.ToLower(constraint)
	switch {
	case strings.Contains(constraint, "email"):
		return "email"
	case strings.Contains(constraint, "student"):
		return "student_id"
	case strings.Contains(constraint, "exam") && strings.Contains(constraint, "student"):
		return "student_id"
	case strings.Contains(constraint, "code"), strings.Contains(constraint, "token"):
		return "code"
	case strings.Contains(constraint, "domain"):
		return "domain"
	case strings.Contains(constraint, "date"):
		return "date"
	case strings.Contains(constraint, "question_tag"):
		return "question_tag_id"
	case strings.Contains(constraint, "question"):
		return "question_id"
	default:
		return ""
	}
}

func jsonFieldName(field string) string {
	if field == "" {
		return ""
	}
	var out []rune
	for index, char := range field {
		if index > 0 && char >= 'A' && char <= 'Z' {
			out = append(out, '_')
		}
		out = append(out, []rune(strings.ToLower(string(char)))...)
	}
	return string(out)
}

func validationMessage(field, tag, param string) string {
	label := labelForField(field)
	switch tag {
	case "required":
		return label + " wajib diisi."
	case "min":
		return label + " minimal " + param + " karakter."
	case "max":
		return label + " maksimal " + param + " karakter."
	case "email":
		return "Format email tidak valid."
	case "oneof":
		return label + " berisi pilihan yang tidak valid."
	case "gte":
		return label + " harus lebih besar atau sama dengan " + param + "."
	case "lte":
		return label + " harus lebih kecil atau sama dengan " + param + "."
	default:
		return label + " tidak valid."
	}
}

func labelForField(field string) string {
	switch field {
	case "code":
		return "Kode"
	case "name":
		return "Nama"
	case "email", "account_email":
		return "Email"
	case "password", "account_password":
		return "Password"
	case "student_id":
		return "Siswa"
	case "class_room_id":
		return "Kelas"
	case "question_tag_id":
		return "Tag soal"
	case "question_id":
		return "Soal"
	case "domain":
		return "Domain"
	case "date":
		return "Tanggal"
	default:
		if field == "" {
			return "Data"
		}
		return strings.Title(strings.ReplaceAll(field, "_", " "))
	}
}
