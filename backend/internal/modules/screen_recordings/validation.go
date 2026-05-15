package screen_recordings

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateScreenRecordingRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateScreenRecordingRequest) error { return validate.Struct(req) }
