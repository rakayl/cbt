package subscription_plans

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type SubscriptionPlan struct{ shared.Record }

const TableName = "subscription_plans"
const PermissionRead = "subscription.plans:read"
const PermissionWrite = "subscription.plans:write"
