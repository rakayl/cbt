package answers

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Answer struct{ shared.Record }

const TableName = "answers"
const PermissionRead = "answers:read"
const PermissionWrite = "answers:write"
