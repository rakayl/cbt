package reconnect_logs

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type ReconnectLog struct{ shared.Record }

const TableName = "reconnect_logs"
const PermissionRead = "reconnect.logs:read"
const PermissionWrite = "reconnect.logs:write"
