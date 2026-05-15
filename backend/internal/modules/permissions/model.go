package permissions

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Permission struct{ shared.Record }

const TableName = "permissions"
const PermissionRead = "permissions:read"
const PermissionWrite = "permissions:write"
