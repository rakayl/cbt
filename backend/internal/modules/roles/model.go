package roles

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Role struct{ shared.Record }

const TableName = "roles"
const PermissionRead = "roles:read"
const PermissionWrite = "roles:write"
