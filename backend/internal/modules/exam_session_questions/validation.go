package exam_session_questions

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateExamSessionQuestionRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateExamSessionQuestionRequest) error { return validate.Struct(req) }
