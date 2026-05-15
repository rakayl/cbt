package grading

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateGradingRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateGradingRequest) error { return validate.Struct(req) }
