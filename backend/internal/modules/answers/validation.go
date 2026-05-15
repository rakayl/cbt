package answers

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateAnswerRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateAnswerRequest) error { return validate.Struct(req) }
