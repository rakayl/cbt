package enrollment

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateEnrollmentRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateEnrollmentRequest) error { return validate.Struct(req) }
