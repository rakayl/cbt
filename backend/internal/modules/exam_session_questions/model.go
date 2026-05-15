package exam_session_questions

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type ExamSessionQuestion struct{ shared.Record }

const TableName = "exam_session_questions"
const PermissionRead = "exam.session.questions:read"
const PermissionWrite = "exam.session.questions:write"
