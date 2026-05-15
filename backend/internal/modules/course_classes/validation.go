package course_classes

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateCourseClasseRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateCourseClasseRequest) error { return validate.Struct(req) }
