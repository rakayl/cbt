package enrollment

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Enrollment struct{ shared.Record }

const TableName = "enrollment"
const PermissionRead = "enrollment:read"
const PermissionWrite = "enrollment:write"
