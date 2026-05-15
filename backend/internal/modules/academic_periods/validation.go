package academic_periods

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateAcademicPeriodRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateAcademicPeriodRequest) error { return validate.Struct(req) }
