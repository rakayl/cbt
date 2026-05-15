package question_tags

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateQuestionTagRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateQuestionTagRequest) error { return validate.Struct(req) }
