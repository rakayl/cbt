package question_tags

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type QuestionTag struct{ shared.Record }

const TableName = "question_tags"
const PermissionRead = "question.tags:read"
const PermissionWrite = "question.tags:write"
