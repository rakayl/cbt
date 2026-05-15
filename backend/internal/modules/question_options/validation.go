package question_options

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateQuestionOptionRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateQuestionOptionRequest) error { return validate.Struct(req) }
