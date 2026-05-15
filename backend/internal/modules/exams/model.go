package exams

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Exam struct{ shared.Record }

const TableName = "exams"
const PermissionRead = "exams:read"
const PermissionWrite = "exams:write"
