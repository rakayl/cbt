package proctoring

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Proctoring struct{ shared.Record }

const TableName = "proctoring"
const PermissionRead = "proctoring:read"
const PermissionWrite = "proctoring:write"
