package notifications

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateNotificationRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateNotificationRequest) error { return validate.Struct(req) }
