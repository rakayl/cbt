package course_classes

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type CourseClasse struct{ shared.Record }

const TableName = "course_classes"
const PermissionRead = "course.classes:read"
const PermissionWrite = "course.classes:write"
