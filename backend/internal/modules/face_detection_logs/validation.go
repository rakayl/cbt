package face_detection_logs

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateFaceDetectionLogRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateFaceDetectionLogRequest) error { return validate.Struct(req) }
