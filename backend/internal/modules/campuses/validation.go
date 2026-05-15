package campuses

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateCampuseRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateCampuseRequest) error { return validate.Struct(req) }
