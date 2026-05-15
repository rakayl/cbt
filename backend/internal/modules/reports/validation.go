package reports

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateReportRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateReportRequest) error { return validate.Struct(req) }
