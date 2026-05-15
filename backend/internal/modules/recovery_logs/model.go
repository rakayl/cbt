package recovery_logs

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type RecoveryLog struct{ shared.Record }

const TableName = "recovery_logs"
const PermissionRead = "recovery.logs:read"
const PermissionWrite = "recovery.logs:write"
