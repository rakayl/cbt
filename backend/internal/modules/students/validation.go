package students

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateStudentRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateStudentRequest) error { return validate.Struct(req) }
