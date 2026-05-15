package question_banks

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type QuestionBank struct{ shared.Record }

const TableName = "question_banks"
const PermissionRead = "question.banks:read"
const PermissionWrite = "question.banks:write"
