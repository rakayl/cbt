package permissions

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreatePermissionRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdatePermissionRequest) error { return validate.Struct(req) }
