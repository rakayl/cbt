package audit_logs

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type AuditLog struct{ shared.Record }

const TableName = "audit_logs"
const PermissionRead = "audit.logs:read"
const PermissionWrite = "audit.logs:write"
