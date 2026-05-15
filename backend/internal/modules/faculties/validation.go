package faculties

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateFacultyRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateFacultyRequest) error { return validate.Struct(req) }
