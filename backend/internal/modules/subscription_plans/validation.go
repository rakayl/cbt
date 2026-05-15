package subscription_plans

import "github.com/go-playground/validator/v10"

var validate = validator.New()

func ValidateCreate(req CreateSubscriptionPlanRequest) error { return validate.Struct(req) }
func ValidateUpdate(req UpdateSubscriptionPlanRequest) error { return validate.Struct(req) }
