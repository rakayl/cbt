package roles

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateRoleRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateRoleRequest) error { return validate.Struct(req) }
