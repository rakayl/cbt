package tenants

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Tenant struct{ shared.Record }

const TableName = "tenants"
const PermissionRead = "tenants:read"
const PermissionWrite = "tenants:write"
