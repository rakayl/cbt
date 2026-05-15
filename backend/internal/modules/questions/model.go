package questions

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Question struct{ shared.Record }

const TableName = "questions"
const PermissionRead = "questions:read"
const PermissionWrite = "questions:write"
