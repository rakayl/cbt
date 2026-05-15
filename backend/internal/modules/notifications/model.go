package notifications

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Notification struct{ shared.Record }

const TableName = "notifications"
const PermissionRead = "notifications:read"
const PermissionWrite = "notifications:write"
