package auth

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Auth struct{ shared.Record }

const TableName = "auth"
const PermissionRead = "auth:read"
const PermissionWrite = "auth:write"
