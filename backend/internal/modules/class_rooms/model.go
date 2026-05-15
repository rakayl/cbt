package class_rooms

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type ClassRoom struct{ shared.Record }

const TableName = "class_rooms"
const PermissionRead = "class.rooms:read"
const PermissionWrite = "class.rooms:write"
