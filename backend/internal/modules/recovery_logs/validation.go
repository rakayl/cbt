package recovery_logs

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateRecoveryLogRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateRecoveryLogRequest) error { return validate.Struct(req) }
