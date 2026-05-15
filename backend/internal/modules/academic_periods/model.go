package academic_periods

import "github.com/cbt-ai/enterprise-cbt/internal/shared"

type AcademicPeriod struct{ shared.Record }

const TableName = "academic_periods"
const PermissionRead = "academic.periods:read"
const PermissionWrite = "academic.periods:write"
