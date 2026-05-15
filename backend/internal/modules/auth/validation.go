package auth

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateAuthRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateAuthRequest) error { return validate.Struct(req) }
