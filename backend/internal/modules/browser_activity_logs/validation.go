package browser_activity_logs

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateBrowserActivityLogRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateBrowserActivityLogRequest) error { return validate.Struct(req) }
