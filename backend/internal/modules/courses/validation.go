package courses

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateCourseRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateCourseRequest) error { return validate.Struct(req) }
