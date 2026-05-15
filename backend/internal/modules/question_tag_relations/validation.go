package question_tag_relations

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateQuestionTagRelationRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateQuestionTagRelationRequest) error { return validate.Struct(req) }
