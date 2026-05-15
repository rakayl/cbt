package analytics

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateAnalyticRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateAnalyticRequest) error { return validate.Struct(req) }
