package users

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type User struct{ shared.Record }

const TableName = "users"
const PermissionRead = "users:read"
const PermissionWrite = "users:write"
