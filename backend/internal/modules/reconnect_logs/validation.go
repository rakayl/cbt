package reconnect_logs

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateReconnectLogRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateReconnectLogRequest) error { return validate.Struct(req) }
