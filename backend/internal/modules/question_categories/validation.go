package question_categories

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateQuestionCategoryRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateQuestionCategoryRequest) error { return validate.Struct(req) }
