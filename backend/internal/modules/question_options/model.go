package question_options

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type QuestionOption struct{ shared.Record }

const TableName = "question_options"
const PermissionRead = "question.options:read"
const PermissionWrite = "question.options:write"
