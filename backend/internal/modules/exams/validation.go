package exams

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateExamRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateExamRequest) error { return validate.Struct(req) }
func ValidatePublish(req PublishExamRequest) error {
	return validate.Struct(req)
}
