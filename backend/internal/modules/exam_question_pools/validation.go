package exam_question_pools

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateExamQuestionPoolRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateExamQuestionPoolRequest) error { return validate.Struct(req) }
