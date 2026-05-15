package courses

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Course struct{ shared.Record }

const TableName = "courses"
const PermissionRead = "courses:read"
const PermissionWrite = "courses:write"
