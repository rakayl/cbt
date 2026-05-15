package faculties

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Faculty struct{ shared.Record }

const TableName = "faculties"
const PermissionRead = "faculties:read"
const PermissionWrite = "faculties:write"
