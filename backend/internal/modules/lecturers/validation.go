package lecturers

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateLecturerRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateLecturerRequest) error { return validate.Struct(req) }
