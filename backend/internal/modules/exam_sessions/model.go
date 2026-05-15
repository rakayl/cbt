package exam_sessions

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type ExamSession struct{ shared.Record }

const TableName = "exam_sessions"
const PermissionRead = "exam.sessions:read"
const PermissionWrite = "exam.sessions:write"
