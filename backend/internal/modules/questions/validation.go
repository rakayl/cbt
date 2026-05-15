package questions

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateQuestionRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateQuestionRequest) error { return validate.Struct(req) }
