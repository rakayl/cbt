package proctoring

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateProctoringRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateProctoringRequest) error { return validate.Struct(req) }
