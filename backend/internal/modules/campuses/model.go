package campuses

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Campuse struct{ shared.Record }

const TableName = "campuses"
const PermissionRead = "campuses:read"
const PermissionWrite = "campuses:write"
