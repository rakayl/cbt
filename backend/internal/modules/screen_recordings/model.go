package screen_recordings

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type ScreenRecording struct{ shared.Record }

const TableName = "screen_recordings"
const PermissionRead = "screen.recordings:read"
const PermissionWrite = "screen.recordings:write"
