package grading

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Grading struct{ shared.Record }

const TableName = "grading"
const PermissionRead = "grading:read"
const PermissionWrite = "grading:write"
