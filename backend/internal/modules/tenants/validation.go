package tenants

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateTenantRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateTenantRequest) error { return validate.Struct(req) }
