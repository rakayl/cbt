package question_categories

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type QuestionCategory struct{ shared.Record }

const TableName = "question_categories"
const PermissionRead = "question.categories:read"
const PermissionWrite = "question.categories:write"
