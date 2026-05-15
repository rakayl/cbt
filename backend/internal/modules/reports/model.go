package reports

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Report struct{ shared.Record }

const TableName = "reports"
const PermissionRead = "reports:read"
const PermissionWrite = "reports:write"
