package analytics

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Analytic struct{ shared.Record }

const TableName = "analytics"
const PermissionRead = "analytics:read"
const PermissionWrite = "analytics:write"
