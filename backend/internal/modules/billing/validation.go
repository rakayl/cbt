package billing

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateBillingRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateBillingRequest) error { return validate.Struct(req) }
