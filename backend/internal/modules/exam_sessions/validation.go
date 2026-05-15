package exam_sessions

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateExamSessionRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateExamSessionRequest) error { return validate.Struct(req) }
