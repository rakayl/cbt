package class_rooms

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateClassRoomRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateClassRoomRequest) error { return validate.Struct(req) }
