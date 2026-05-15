package study_programs

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type StudyProgram struct{ shared.Record }

const TableName = "study_programs"
const PermissionRead = "study.programs:read"
const PermissionWrite = "study.programs:write"
