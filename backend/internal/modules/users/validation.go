package users

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateUserRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateUserRequest) error { return validate.Struct(req) }
