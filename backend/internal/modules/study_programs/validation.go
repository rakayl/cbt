package study_programs

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateStudyProgramRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateStudyProgramRequest) error { return validate.Struct(req) }
