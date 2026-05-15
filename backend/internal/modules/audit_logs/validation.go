package audit_logs

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateAuditLogRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateAuditLogRequest) error { return validate.Struct(req) }
