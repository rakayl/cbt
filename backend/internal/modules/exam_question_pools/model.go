package exam_question_pools

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type ExamQuestionPool struct{ shared.Record }

const TableName = "exam_question_pools"
const PermissionRead = "exam.question.pools:read"
const PermissionWrite = "exam.question.pools:write"
