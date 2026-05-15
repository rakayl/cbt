package browser_activity_logs

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type BrowserActivityLog struct{ shared.Record }

const TableName = "browser_activity_logs"
const PermissionRead = "browser.activity.logs:read"
const PermissionWrite = "browser.activity.logs:write"
