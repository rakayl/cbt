package question_banks

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateQuestionBankRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateQuestionBankRequest) error { return validate.Struct(req) }
