package billing

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Billing struct{ shared.Record }

const TableName = "billing"
const PermissionRead = "billing:read"
const PermissionWrite = "billing:write"
