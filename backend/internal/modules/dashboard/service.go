package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service interface {
	Summary(context.Context, uuid.UUID, uuid.UUID, []string) (DashboardSummaryResponse, error)
}

type service struct {
	deps shared.Deps
}

func NewService(deps shared.Deps) Service {
	return &service{deps: deps}
}

func (s *service) Summary(ctx context.Context, tenantID, userID uuid.UUID, permissions []string) (DashboardSummaryResponse, error) {
	role := resolveDashboardRole(permissions)
	out := DashboardSummaryResponse{
		Role:        role,
		TenantID:    tenantID,
		GeneratedAt: time.Now().UTC(),
		Metrics:     []MetricView{},
		Health:      map[string]string{},
	}

	switch role {
	case "student":
		return s.studentSummary(ctx, tenantID, userID, out)
	case "lecturer":
		return s.lecturerSummary(ctx, tenantID, userID, out)
	case "proctor":
		return s.proctorSummary(ctx, tenantID, out)
	case "super_admin":
		return s.superAdminSummary(ctx, out)
	default:
		return s.tenantAdminSummary(ctx, tenantID, out)
	}
}

func (s *service) superAdminSummary(ctx context.Context, out DashboardSummaryResponse) (DashboardSummaryResponse, error) {
	tenantCount, err := s.countTable(ctx, "tenants", uuid.Nil, "")
	if err != nil {
		return out, err
	}
	userCount, err := s.countTable(ctx, "users", uuid.Nil, "")
	if err != nil {
		return out, err
	}
	examCount, err := s.countTable(ctx, "exams", uuid.Nil, "")
	if err != nil {
		return out, err
	}
	activeSessions, err := s.countTable(ctx, "exam_sessions", uuid.Nil, "status_enum IN ('started','reconnecting')")
	if err != nil {
		return out, err
	}
	out.Metrics = []MetricView{
		metric("tenants", "Tenant Aktif", tenantCount, "primary"),
		metric("users", "Total User", userCount, "success"),
		metric("exams", "Total Ujian", examCount, "info"),
		metric("active_sessions", "Sesi Aktif", activeSessions, "warning"),
	}
	out.Health = s.health(ctx)
	out.Trends, _ = s.sessionTrends(ctx, uuid.Nil)
	out.Activities, _ = s.recentActivities(ctx, uuid.Nil, 8)
	return out, nil
}

func (s *service) tenantAdminSummary(ctx context.Context, tenantID uuid.UUID, out DashboardSummaryResponse) (DashboardSummaryResponse, error) {
	students, err := s.countTable(ctx, "students", tenantID, "")
	if err != nil {
		return out, err
	}
	lecturers, err := s.countTable(ctx, "lecturers", tenantID, "")
	if err != nil {
		return out, err
	}
	questions, err := s.countTable(ctx, "questions", tenantID, "")
	if err != nil {
		return out, err
	}
	activeSessions, err := s.countTable(ctx, "exam_sessions", tenantID, "status_enum IN ('started','reconnecting')")
	if err != nil {
		return out, err
	}
	draftExams, err := s.countTable(ctx, "exams", tenantID, "status='draft'")
	if err != nil {
		return out, err
	}
	publishedExams, err := s.countTable(ctx, "exams", tenantID, "status IN ('published','active')")
	if err != nil {
		return out, err
	}
	out.Metrics = []MetricView{
		metric("students", "Siswa", students, "primary"),
		metric("lecturers", "Guru", lecturers, "success"),
		metric("questions", "Bank Soal", questions, "info"),
		metric("active_sessions", "Sesi Berjalan", activeSessions, "warning"),
		metric("draft_exams", "Draft Ujian", draftExams, "muted"),
		metric("published_exams", "Published", publishedExams, "success"),
	}
	out.ActiveExams, _ = s.activeExams(ctx, tenantID, uuid.Nil)
	out.Activities, _ = s.recentActivities(ctx, tenantID, 8)
	out.Trends, _ = s.sessionTrends(ctx, tenantID)
	out.Health = s.health(ctx)
	return out, nil
}

func (s *service) lecturerSummary(ctx context.Context, tenantID, userID uuid.UUID, out DashboardSummaryResponse) (DashboardSummaryResponse, error) {
	questions, err := s.countTable(ctx, "questions", tenantID, fmt.Sprintf("owner_user_id='%s'", userID))
	if err != nil {
		return out, err
	}
	tags, err := s.countTable(ctx, "question_tags", tenantID, fmt.Sprintf("owner_user_id='%s'", userID))
	if err != nil {
		return out, err
	}
	exams, err := s.countTable(ctx, "exams", tenantID, fmt.Sprintf("owner_user_id='%s'", userID))
	if err != nil {
		return out, err
	}
	activeSessions, err := s.countOwnedSessions(ctx, tenantID, userID, "es.status_enum IN ('started','reconnecting')")
	if err != nil {
		return out, err
	}
	pendingManual, err := s.countOwnedManualReview(ctx, tenantID, userID)
	if err != nil {
		return out, err
	}
	out.Metrics = []MetricView{
		metric("questions", "Soal Saya", questions, "primary"),
		metric("question_tags", "Tag/Jenis Soal", tags, "info"),
		metric("exams", "Ujian Saya", exams, "success"),
		metric("active_sessions", "Sesi Berjalan", activeSessions, "warning"),
		metric("manual_review", "Perlu Grading", pendingManual, "danger"),
	}
	out.ActiveExams, _ = s.activeExams(ctx, tenantID, userID)
	out.Activities, _ = s.recentActivities(ctx, tenantID, 8)
	out.Trends, _ = s.sessionTrends(ctx, tenantID)
	return out, nil
}

func (s *service) proctorSummary(ctx context.Context, tenantID uuid.UUID, out DashboardSummaryResponse) (DashboardSummaryResponse, error) {
	activeSessions, err := s.countTable(ctx, "exam_sessions", tenantID, "status_enum IN ('started','reconnecting')")
	if err != nil {
		return out, err
	}
	reconnecting, err := s.countTable(ctx, "exam_sessions", tenantID, "status_enum='reconnecting'")
	if err != nil {
		return out, err
	}
	highRisk, err := s.countTable(ctx, "proctoring_logs", tenantID, "score >= 50")
	if err != nil {
		return out, err
	}
	todayEvents, err := s.countTable(ctx, "proctoring_logs", tenantID, "created_at >= current_date")
	if err != nil {
		return out, err
	}
	out.Metrics = []MetricView{
		metric("active_sessions", "Peserta Online", activeSessions, "primary"),
		metric("reconnecting", "Reconnect", reconnecting, "warning"),
		metric("high_risk", "High Risk", highRisk, "danger"),
		metric("events_today", "Event Hari Ini", todayEvents, "info"),
	}
	out.ActiveExams, _ = s.activeExams(ctx, tenantID, uuid.Nil)
	out.Activities, _ = s.recentActivities(ctx, tenantID, 12)
	return out, nil
}

func (s *service) studentSummary(ctx context.Context, tenantID, userID uuid.UUID, out DashboardSummaryResponse) (DashboardSummaryResponse, error) {
	studentID, err := s.resolveStudentID(ctx, tenantID, userID)
	if err != nil {
		out.Metrics = []MetricView{
			metric("available_exams", "Ujian Tersedia", 0, "primary"),
			metric("active_sessions", "Sedang Dikerjakan", 0, "warning"),
			metric("completed", "Selesai", 0, "success"),
			metric("classes", "Kelas Diikuti", 0, "info"),
		}
		return out, nil
	}
	available, err := s.countStudentInvites(ctx, tenantID, studentID)
	if err != nil {
		return out, err
	}
	active, err := s.countStudentSessions(ctx, tenantID, studentID, "status_enum IN ('started','reconnecting')")
	if err != nil {
		return out, err
	}
	completed, err := s.countStudentSessions(ctx, tenantID, studentID, "status_enum='completed'")
	if err != nil {
		return out, err
	}
	classes, err := s.countTable(ctx, "enrollment", tenantID, fmt.Sprintf("student_id='%s' AND active=true", studentID))
	if err != nil {
		return out, err
	}
	out.Metrics = []MetricView{
		metric("available_exams", "Ujian Tersedia", available, "primary"),
		metric("active_sessions", "Sedang Dikerjakan", active, "warning"),
		metric("completed", "Ujian Selesai", completed, "success"),
		metric("classes", "Kelas Diikuti", classes, "info"),
	}
	out.UpcomingExams, _ = s.studentUpcomingExams(ctx, tenantID, studentID)
	out.RecentResults, _ = s.studentRecentResults(ctx, tenantID, studentID)
	return out, nil
}

func (s *service) countTable(ctx context.Context, table string, tenantID uuid.UUID, extraWhere string) (int64, error) {
	where := "deleted_at IS NULL"
	args := []any{}
	if tenantID != uuid.Nil {
		args = append(args, tenantID)
		where += fmt.Sprintf(" AND tenant_id=$%d", len(args))
	}
	if strings.TrimSpace(extraWhere) != "" {
		where += " AND " + extraWhere
	}
	var total int64
	err := s.deps.DB.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", table, where), args...).Scan(&total)
	return total, err
}

func (s *service) countOwnedSessions(ctx context.Context, tenantID, ownerUserID uuid.UUID, extra string) (int64, error) {
	where := "es.tenant_id=$1 AND e.owner_user_id=$2 AND es.deleted_at IS NULL AND e.deleted_at IS NULL"
	if strings.TrimSpace(extra) != "" {
		where += " AND " + extra
	}
	var total int64
	err := s.deps.DB.QueryRow(ctx, `
		SELECT count(*)
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id
		WHERE `+where, tenantID, ownerUserID).Scan(&total)
	return total, err
}

func (s *service) countOwnedManualReview(ctx context.Context, tenantID, ownerUserID uuid.UUID) (int64, error) {
	var total int64
	err := s.deps.DB.QueryRow(ctx, `
		SELECT count(*)
		FROM grading g
		JOIN exam_sessions es ON es.id=(g.metadata->>'exam_session_id')::uuid AND es.deleted_at IS NULL
		JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
		WHERE g.tenant_id=$1
		  AND g.deleted_at IS NULL
		  AND e.owner_user_id=$2
		  AND COALESCE(g.metadata->>'manual_status','') NOT IN ('reviewed')
		  AND COALESCE(g.metadata->>'answer_mode','') NOT IN ('single','multiple')`, tenantID, ownerUserID).Scan(&total)
	return total, err
}

func (s *service) countStudentInvites(ctx context.Context, tenantID, studentID uuid.UUID) (int64, error) {
	var total int64
	err := s.deps.DB.QueryRow(ctx, `
		SELECT count(*)
		FROM exam_invites ei
		JOIN exams e ON e.id=ei.exam_id AND e.deleted_at IS NULL
		WHERE ei.tenant_id=$1
		  AND ei.student_id=$2
		  AND ei.deleted_at IS NULL
		  AND e.status IN ('published','active')`, tenantID, studentID).Scan(&total)
	return total, err
}

func (s *service) countStudentSessions(ctx context.Context, tenantID, studentID uuid.UUID, extra string) (int64, error) {
	where := "tenant_id=$1 AND student_id=$2 AND deleted_at IS NULL"
	if strings.TrimSpace(extra) != "" {
		where += " AND " + extra
	}
	var total int64
	err := s.deps.DB.QueryRow(ctx, "SELECT count(*) FROM exam_sessions WHERE "+where, tenantID, studentID).Scan(&total)
	return total, err
}

func (s *service) activeExams(ctx context.Context, tenantID, ownerUserID uuid.UUID) ([]DashboardExamView, error) {
	args := []any{tenantID}
	where := "es.tenant_id=$1 AND es.deleted_at IS NULL AND e.deleted_at IS NULL AND es.status_enum IN ('started','reconnecting')"
	if ownerUserID != uuid.Nil {
		args = append(args, ownerUserID)
		where += fmt.Sprintf(" AND e.owner_user_id=$%d", len(args))
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT e.id, e.code, e.name, e.status, e.published_at, COALESCE(e.metadata,'{}'::jsonb),
		       count(es.id) AS participant_count,
		       count(es.id) FILTER (WHERE es.status_enum='completed') AS completed_count
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id
		WHERE `+where+`
		GROUP BY e.id, e.code, e.name, e.status, e.published_at, e.metadata
		ORDER BY participant_count DESC, e.name ASC
		LIMIT 6`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExamRows(rows)
}

func (s *service) studentUpcomingExams(ctx context.Context, tenantID, studentID uuid.UUID) ([]DashboardExamView, error) {
	rows, err := s.deps.DB.Query(ctx, `
		SELECT e.id, e.code, e.name, e.status, e.published_at, COALESCE(e.metadata,'{}'::jsonb),
		       ei.invitation_code
		FROM exam_invites ei
		JOIN exams e ON e.id=ei.exam_id AND e.deleted_at IS NULL
		WHERE ei.tenant_id=$1
		  AND ei.student_id=$2
		  AND ei.deleted_at IS NULL
		  AND e.status IN ('published','active')
		ORDER BY e.published_at DESC NULLS LAST, ei.created_at DESC
		LIMIT 6`, tenantID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DashboardExamView{}
	for rows.Next() {
		var item DashboardExamView
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Status, &item.PublishedAt, &raw, &item.InvitationCode); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(raw, &item.Metadata)
		item.StartsAt = timePtrFromMetadata(item.Metadata, "starts_at")
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) studentRecentResults(ctx context.Context, tenantID, studentID uuid.UUID) ([]DashboardResultView, error) {
	rows, err := s.deps.DB.Query(ctx, `
		SELECT es.id, e.name, es.code, es.status_enum::text, es.submitted_at, COALESCE(es.metadata,'{}'::jsonb)
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
		WHERE es.tenant_id=$1 AND es.student_id=$2 AND es.deleted_at IS NULL
		ORDER BY es.created_at DESC
		LIMIT 6`, tenantID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DashboardResultView{}
	for rows.Next() {
		var item DashboardResultView
		var raw []byte
		if err := rows.Scan(&item.SessionID, &item.ExamName, &item.Code, &item.Status, &item.SubmittedAt, &raw); err != nil {
			return nil, err
		}
		meta := map[string]any{}
		_ = json.Unmarshal(raw, &meta)
		item.Score = numberFromAny(meta["score"])
		item.MaxScore = numberFromAny(meta["max_score"])
		item.Percentage = numberFromAny(meta["percentage"])
		item.Passed = boolFromAny(meta["passed"])
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) recentActivities(ctx context.Context, tenantID uuid.UUID, limit int) ([]DashboardActivityView, error) {
	args := []any{}
	where := "deleted_at IS NULL"
	if tenantID != uuid.Nil {
		args = append(args, tenantID)
		where += fmt.Sprintf(" AND tenant_id=$%d", len(args))
	}
	args = append(args, limit)
	rows, err := s.deps.DB.Query(ctx, `
		SELECT id, COALESCE(event_type, name, 'proctoring_event'), COALESCE(description,''), COALESCE(score,0), created_at, COALESCE(metadata,'{}'::jsonb)
		FROM proctoring_logs
		WHERE `+where+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DashboardActivityView{}
	for rows.Next() {
		var item DashboardActivityView
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Title, &item.Subtitle, &item.Score, &item.CreatedAt, &raw); err != nil {
			return nil, err
		}
		item.Type = "proctoring"
		item.Severity = severityFromScore(item.Score)
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(raw, &item.Metadata)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) sessionTrends(ctx context.Context, tenantID uuid.UUID) ([]DashboardTrendPoint, error) {
	args := []any{}
	where := "es.deleted_at IS NULL AND es.created_at >= current_date - interval '6 days'"
	if tenantID != uuid.Nil {
		args = append(args, tenantID)
		where += fmt.Sprintf(" AND es.tenant_id=$%d", len(args))
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT to_char(day::date, 'YYYY-MM-DD'), COALESCE(count(es.id),0)
		FROM generate_series(current_date - interval '6 days', current_date, interval '1 day') day
		LEFT JOIN exam_sessions es ON date_trunc('day', es.created_at)=day AND `+where+`
		GROUP BY day
		ORDER BY day ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DashboardTrendPoint{}
	for rows.Next() {
		var item DashboardTrendPoint
		if err := rows.Scan(&item.Date, &item.Value); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) resolveStudentID(ctx context.Context, tenantID, userID uuid.UUID) (uuid.UUID, error) {
	var studentID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `SELECT id FROM students WHERE tenant_id=$1 AND user_id=$2 AND deleted_at IS NULL LIMIT 1`, tenantID, userID).Scan(&studentID)
	return studentID, err
}

func (s *service) health(ctx context.Context) map[string]string {
	out := map[string]string{
		"postgres": "unknown",
		"redis":    "disabled",
		"rabbitmq": "disabled",
		"storage":  "disabled",
	}
	if err := s.deps.DB.Ping(ctx); err == nil {
		out["postgres"] = "healthy"
	} else {
		out["postgres"] = "unhealthy"
	}
	if s.deps.Redis != nil {
		if err := s.deps.Redis.Ping(ctx).Err(); err == nil {
			out["redis"] = "healthy"
		} else {
			out["redis"] = "unhealthy"
		}
	}
	if s.deps.Rabbit != nil {
		out["rabbitmq"] = "configured"
	}
	if s.deps.Storage != nil {
		out["storage"] = "configured"
	}
	return out
}

func scanExamRows(rows pgx.Rows) ([]DashboardExamView, error) {
	out := []DashboardExamView{}
	for rows.Next() {
		var item DashboardExamView
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Status, &item.PublishedAt, &raw, &item.ParticipantCnt, &item.CompletedCnt); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(raw, &item.Metadata)
		item.StartsAt = timePtrFromMetadata(item.Metadata, "starts_at")
		out = append(out, item)
	}
	return out, rows.Err()
}

func resolveDashboardRole(permissions []string) string {
	if hasPermission(permissions, "*") {
		return "super_admin"
	}
	if hasPermission(permissions, "exams:join") && !hasPermission(permissions, "exams:write") {
		return "student"
	}
	if hasPermission(permissions, "exams:invite") || hasPermission(permissions, "question.banks:write") {
		return "lecturer"
	}
	if hasPermission(permissions, "proctoring:read") {
		return "proctor"
	}
	return "tenant_admin"
}

func hasPermission(permissions []string, permission string) bool {
	for _, item := range permissions {
		if item == permission || item == "*" {
			return true
		}
	}
	return false
}

func metric(key, label string, value int64, tone string) MetricView {
	return MetricView{Key: key, Label: label, Value: float64(value), Display: formatCompact(value), Tone: tone}
}

func formatCompact(value int64) string {
	if value < 1000 {
		return fmt.Sprint(value)
	}
	if value < 1000000 {
		return fmt.Sprintf("%.1fK", float64(value)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(value)/1000000)
}

func timePtrFromMetadata(meta map[string]any, key string) *time.Time {
	raw := strings.TrimSpace(fmt.Sprint(meta[key]))
	if raw == "" || raw == "<nil>" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil
	}
	return &parsed
}

func severityFromScore(score float64) string {
	if score >= 80 {
		return "critical"
	}
	if score >= 50 {
		return "high"
	}
	if score >= 20 {
		return "medium"
	}
	return "low"
}

func numberFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		out, _ := typed.Float64()
		return out
	case string:
		parsed, _ := strconv.ParseFloat(typed, 64)
		return parsed
	default:
		return 0
	}
}

func boolFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true" || typed == "1"
	default:
		return false
	}
}
