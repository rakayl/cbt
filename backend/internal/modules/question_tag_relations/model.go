package question_tag_relations

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type QuestionTagRelation struct{ shared.Record }

const TableName = "question_tag_relations"
const PermissionRead = "question.tag.relations:read"
const PermissionWrite = "question.tag.relations:write"
