package lecturers

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type Lecturer struct{ shared.Record }

const TableName = "lecturers"
const PermissionRead = "lecturers:read"
const PermissionWrite = "lecturers:write"
