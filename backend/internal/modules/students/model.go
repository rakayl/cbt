package students

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Student struct{ shared.Record }

const TableName = "students"
const PermissionRead = "students:read"
const PermissionWrite = "students:write"
