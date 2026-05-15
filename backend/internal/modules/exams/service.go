package exams

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service interface {
	List(context.Context, uuid.UUID, pagination.Query) (shared.ListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (shared.Record, error)
	Create(context.Context, uuid.UUID, uuid.UUID, CreateExamRequest) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateExamRequest) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	CreateRevision(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) (ExamRevisionResponse, error)
	ListRevisions(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) (ExamRevisionListResult, error)
	Publish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string, PublishExamRequest) (PublishExamResponse, error)
	InviteStudents(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string, InviteStudentsRequest) ([]ExamInviteView, error)
	ListInvites(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) ([]ExamInviteView, error)
	ShareCode(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) (ShareCodeResponse, error)
	JoinByCode(context.Context, uuid.UUID, uuid.UUID, JoinByCodeRequest) (JoinByCodeResponse, error)
	StudentExams(context.Context, uuid.UUID, uuid.UUID, pagination.Query) (StudentExamListResult, error)
}
type service struct {
	repo Repository
	deps shared.Deps
}

func NewService(repo Repository, deps shared.Deps) Service { return &service{repo: repo, deps: deps} }
func (s *service) List(ctx context.Context, tenantID uuid.UUID, q pagination.Query) (shared.ListResult, error) {
	return s.repo.List(ctx, tenantID, q)
}
func (s *service) Get(ctx context.Context, tenantID, id uuid.UUID) (shared.Record, error) {
	return s.repo.Get(ctx, tenantID, id)
}
func (s *service) Create(ctx context.Context, tenantID, userID uuid.UUID, req CreateExamRequest) (shared.Record, error) {
	if err := ValidateCreate(req); err != nil {
		return shared.Record{}, err
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = "draft"
	}
	settings := settingsFromCreate(req)
	if settings.ExamToken == "" {
		if token, ok := req.Metadata["exam_token"].(string); ok {
			settings.ExamToken = strings.TrimSpace(token)
		}
	}
	if settings.ExamToken == "" {
		settings.ExamToken = newCode("EXM", 10)
	}
	req.Metadata["owner_user_id"] = userID.String()
	mergeExamMetadata(req.Metadata, settings)
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Create(ctx, tenantID, rec)
	if err == nil {
		err = s.applyExamSettings(ctx, tenantID, out.ID, userID, settings, req.Status)
	}
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "exams.created", body)
	}
	return out, err
}
func (s *service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateExamRequest) (shared.Record, error) {
	if err := ValidateUpdate(req); err != nil {
		return shared.Record{}, err
	}
	var currentStatus string
	if err := s.deps.DB.QueryRow(ctx, `SELECT status FROM exams WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, id, tenantID).Scan(&currentStatus); err != nil {
		return shared.Record{}, err
	}
	if currentStatus == "published" || currentStatus == "completed" {
		return shared.Record{}, errors.New("published exams are locked; create a revision draft instead")
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	settings := settingsFromUpdate(req)
	if settings.ExamToken == "" {
		settings.ExamToken = newCode("EXM", 10)
	}
	mergeExamMetadata(req.Metadata, settings)
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Update(ctx, tenantID, id, rec)
	if err == nil {
		err = s.applyExamSettings(ctx, tenantID, id, uuid.Nil, settings, req.Status)
	}
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "exams.updated", body)
	}
	return out, err
}

func (s *service) CreateRevision(ctx context.Context, tenantID, examID, userID uuid.UUID, permissions []string) (ExamRevisionResponse, error) {
	if err := s.ensureExamWritable(ctx, tenantID, examID, userID, permissions); err != nil {
		return ExamRevisionResponse{}, err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return ExamRevisionResponse{}, err
	}
	defer tx.Rollback(ctx)

	var source shared.Record
	var rawMetadata []byte
	var revisionNumber int
	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, code, name, COALESCE(description,''), status, COALESCE(metadata,'{}'::jsonb),
		       COALESCE((metadata->>'revision_number')::int, 1) + 1
		FROM exams
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL
		FOR UPDATE`,
		examID, tenantID).Scan(&source.ID, &source.TenantID, &source.Code, &source.Name, &source.Description, &source.Status, &rawMetadata, &revisionNumber)
	if err != nil {
		return ExamRevisionResponse{}, err
	}
	if source.Status != "published" && source.Status != "completed" {
		return ExamRevisionResponse{}, errors.New("only published or completed exams need revision drafts")
	}
	metadata := map[string]any{}
	_ = json.Unmarshal(rawMetadata, &metadata)
	metadata["revision_of_exam_id"] = examID.String()
	metadata["revision_number"] = revisionNumber
	metadata["revision_created_by"] = userID.String()
	metadata["revision_created_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	delete(metadata, "publish_blueprint_snapshot")
	delete(metadata, "publish_blueprint_version")
	newMetadata, _ := json.Marshal(metadata)
	newExamID := uuid.New()
	newCode := fmt.Sprintf("%s-R%d", strings.TrimSpace(source.Code), revisionNumber)
	if len(newCode) > 80 {
		newCode = fmt.Sprintf("REV-%s-%d", examID.String()[:8], revisionNumber)
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO exams(id,tenant_id,code,name,description,status,metadata,owner_user_id,course_class_id,exam_token,duration_minutes,passing_grade,random_question,random_option,max_attempt,instruction)
		SELECT $1, tenant_id, $2, $3, description, 'draft', $4, owner_user_id, course_class_id, $5, duration_minutes, passing_grade, random_question, random_option, max_attempt, instruction
		FROM exams
		WHERE id=$6 AND tenant_id=$7 AND deleted_at IS NULL
		RETURNING id`,
		newExamID, newCode, source.Name+" Revision "+fmt.Sprint(revisionNumber), newMetadata, newCode, examID, tenantID).Scan(&newExamID)
	if err != nil {
		return ExamRevisionResponse{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO exam_question_pools(id,tenant_id,code,name,status,metadata,exam_id,question_bank_id,question_tag_id,question_count)
		SELECT gen_random_uuid(), tenant_id, code || '-R' || $1::text, name, status, metadata, $2, question_bank_id, question_tag_id, question_count
		FROM exam_question_pools
		WHERE tenant_id=$3 AND exam_id=$4 AND deleted_at IS NULL`,
		fmt.Sprint(revisionNumber), newExamID, tenantID, examID)
	if err != nil {
		return ExamRevisionResponse{}, err
	}
	if err := insertExamAudit(ctx, tx, tenantID, userID, "exam.revision_create", examID, map[string]any{
		"revision_exam_id": newExamID.String(),
		"revision_number":  revisionNumber,
	}); err != nil {
		return ExamRevisionResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ExamRevisionResponse{}, err
	}
	out := ExamRevisionResponse{SourceExamID: examID, RevisionExamID: newExamID, RevisionNumber: revisionNumber, Status: "draft"}
	if s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "exams.revision.created", body)
	}
	return out, nil
}

func (s *service) ListRevisions(ctx context.Context, tenantID, examID, userID uuid.UUID, permissions []string) (ExamRevisionListResult, error) {
	if err := s.ensureExamWritable(ctx, tenantID, examID, userID, permissions); err != nil {
		return ExamRevisionListResult{}, err
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT id, code, name, status, COALESCE((metadata->>'revision_number')::int, 1), COALESCE(metadata,'{}'::jsonb), created_at
		FROM exams
		WHERE tenant_id=$1
		  AND deleted_at IS NULL
		  AND metadata->>'revision_of_exam_id'=$2
		ORDER BY COALESCE((metadata->>'revision_number')::int, 1) DESC, created_at DESC`,
		tenantID, examID.String())
	if err != nil {
		return ExamRevisionListResult{}, err
	}
	defer rows.Close()
	out := ExamRevisionListResult{Items: []ExamRevisionView{}}
	for rows.Next() {
		var item ExamRevisionView
		var raw []byte
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Status, &item.RevisionNumber, &raw, &createdAt); err != nil {
			return ExamRevisionListResult{}, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(raw, &item.Metadata)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out.Items = append(out.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ExamRevisionListResult{}, err
	}
	out.Total = len(out.Items)
	return out, nil
}

func (s *service) Publish(ctx context.Context, tenantID, examID, userID uuid.UUID, permissions []string, req PublishExamRequest) (PublishExamResponse, error) {
	if err := ValidatePublish(req); err != nil {
		return PublishExamResponse{}, err
	}
	if err := s.ensureExamWritable(ctx, tenantID, examID, userID, permissions); err != nil {
		return PublishExamResponse{}, err
	}
	current, err := s.currentExamSettings(ctx, tenantID, examID)
	if err != nil {
		return PublishExamResponse{}, err
	}
	settings := settingsFromPublish(req, current)
	if settings.ExamToken == "" {
		settings.ExamToken = newCode("EXM", 10)
	}
	if len(settings.QuestionPools) > 0 {
		total, err := s.validateQuestionPools(ctx, tenantID, userID, permissions, settings.QuestionPools)
		if err != nil {
			return PublishExamResponse{}, err
		}
		settings.QuestionCount = total
	}
	blueprint := s.publishBlueprintSnapshot(ctx, tenantID, examID, userID, settings)

	var publishedAt time.Time
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return PublishExamResponse{}, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `
		UPDATE exams
		SET status='published',
		    exam_token=$1,
		    course_class_id=$2,
		    duration_minutes=$3,
		    passing_grade=$4,
		    random_question=$5,
		    random_option=$6,
		    max_attempt=$7,
		    instruction=$8,
		    published_at=COALESCE(published_at, now()),
		    metadata=COALESCE(metadata, '{}'::jsonb) || $9::jsonb || $12::jsonb || $13::jsonb,
		    updated_at=now()
		WHERE id=$10 AND tenant_id=$11 AND deleted_at IS NULL
		RETURNING published_at`,
		settings.ExamToken,
		settings.CourseClassID,
		settings.DurationMinutes,
		settings.PassingGrade,
		settings.RandomQuestion,
		settings.RandomOption,
		settings.MaxAttempt,
		settings.Instruction,
		mustJSON(examSettingsMetadata(settings)),
		examID,
		tenantID,
		mustJSON(req.Metadata),
		mustJSON(map[string]any{"publish_blueprint_version": 1, "publish_blueprint_snapshot": blueprint}),
	).Scan(&publishedAt)
	if err != nil {
		return PublishExamResponse{}, err
	}
	if len(settings.QuestionPools) > 0 {
		if err := s.replaceQuestionPools(ctx, tx, tenantID, examID, settings.QuestionPools); err != nil {
			return PublishExamResponse{}, err
		}
	}
	if err := insertExamAudit(ctx, tx, tenantID, userID, "exam.publish", examID, map[string]any{"blueprint": blueprint}); err != nil {
		return PublishExamResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PublishExamResponse{}, err
	}
	out := PublishExamResponse{
		ID:              examID,
		ExamToken:       settings.ExamToken,
		Status:          "published",
		PublishedAt:     publishedAt.UTC().Format(time.RFC3339),
		DurationMinutes: settings.DurationMinutes,
		PassingGrade:    settings.PassingGrade,
		QuestionCount:   settings.QuestionCount,
		QuestionPools:   s.questionPoolViews(ctx, tenantID, settings.QuestionPools),
		MaxAttempt:      settings.MaxAttempt,
	}
	if s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "exams.published", body)
	}
	return out, nil
}

func (s *service) InviteStudents(ctx context.Context, tenantID, examID, invitedBy uuid.UUID, permissions []string, req InviteStudentsRequest) ([]ExamInviteView, error) {
	if err := validate.Struct(req); err != nil {
		return nil, err
	}
	if err := s.ensureExamWritable(ctx, tenantID, examID, invitedBy, permissions); err != nil {
		return nil, err
	}
	var examStatus string
	if err := s.deps.DB.QueryRow(ctx, `SELECT status FROM exams WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, examID, tenantID).Scan(&examStatus); err != nil {
		return nil, err
	}
	if examStatus != "published" {
		return nil, errors.New("exam must be published before inviting students")
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := []ExamInviteView{}
	seen := map[uuid.UUID]bool{}
	for _, studentID := range req.StudentIDs {
		if seen[studentID] {
			continue
		}
		seen[studentID] = true
		var studentCode, studentName string
		if err = tx.QueryRow(ctx, `SELECT code, name FROM students WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, studentID, tenantID).Scan(&studentCode, &studentName); err != nil {
			return nil, err
		}
		inviteID := uuid.New()
		code := newCode("INV", 10)
		status := "invited"
		err = tx.QueryRow(ctx, `
			INSERT INTO exam_invites(id,tenant_id,exam_id,student_id,invited_by_user_id,invitation_code,status)
			VALUES($1,$2,$3,$4,$5,$6,'invited')
			ON CONFLICT (tenant_id, exam_id, student_id) WHERE deleted_at IS NULL
			DO UPDATE SET invited_by_user_id=excluded.invited_by_user_id, status='invited', updated_at=now()
			RETURNING id, invitation_code, status`,
			inviteID, tenantID, examID, studentID, invitedBy, code).Scan(&inviteID, &code, &status)
		if err != nil {
			return nil, err
		}
		out = append(out, ExamInviteView{ID: inviteID, ExamID: examID, StudentID: studentID, StudentCode: studentCode, StudentName: studentName, InvitationCode: code, Status: status})
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "exams.invited", body)
	}
	return out, nil
}

func (s *service) ListInvites(ctx context.Context, tenantID, examID, userID uuid.UUID, permissions []string) ([]ExamInviteView, error) {
	if err := s.ensureExamWritable(ctx, tenantID, examID, userID, permissions); err != nil {
		return nil, err
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT ei.id, ei.exam_id, ei.student_id, s.code, s.name, ei.invitation_code, ei.status
		FROM exam_invites ei
		JOIN students s ON s.id=ei.student_id AND s.tenant_id=ei.tenant_id AND s.deleted_at IS NULL
		WHERE ei.tenant_id=$1 AND ei.exam_id=$2 AND ei.deleted_at IS NULL
		ORDER BY ei.created_at DESC`, tenantID, examID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ExamInviteView{}
	for rows.Next() {
		var item ExamInviteView
		if err := rows.Scan(&item.ID, &item.ExamID, &item.StudentID, &item.StudentCode, &item.StudentName, &item.InvitationCode, &item.Status); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) ShareCode(ctx context.Context, tenantID, examID, userID uuid.UUID, permissions []string) (ShareCodeResponse, error) {
	if err := s.ensureExamWritable(ctx, tenantID, examID, userID, permissions); err != nil {
		return ShareCodeResponse{}, err
	}
	code := ""
	if err := s.deps.DB.QueryRow(ctx, `SELECT COALESCE(exam_token,'') FROM exams WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, examID, tenantID).Scan(&code); err != nil {
		return ShareCodeResponse{}, err
	}
	if strings.TrimSpace(code) == "" {
		code = newCode("EXM", 10)
		if _, err := s.deps.DB.Exec(ctx, `UPDATE exams SET exam_token=$1, updated_at=now() WHERE id=$2 AND tenant_id=$3`, code, examID, tenantID); err != nil {
			return ShareCodeResponse{}, err
		}
	}
	return ShareCodeResponse{ExamID: examID, Code: code}, nil
}

func (s *service) JoinByCode(ctx context.Context, tenantID, userID uuid.UUID, req JoinByCodeRequest) (JoinByCodeResponse, error) {
	if err := validate.Struct(req); err != nil {
		return JoinByCodeResponse{}, err
	}
	var studentID uuid.UUID
	if err := s.deps.DB.QueryRow(ctx, `SELECT id FROM students WHERE tenant_id=$1 AND user_id=$2 AND deleted_at IS NULL LIMIT 1`, tenantID, userID).Scan(&studentID); err != nil {
		return JoinByCodeResponse{}, errors.New("student profile not found")
	}
	code := strings.TrimSpace(req.Code)
	out := JoinByCodeResponse{StudentID: studentID}
	err := s.deps.DB.QueryRow(ctx, `
		SELECT e.id, e.name, COALESCE(e.description,''), COALESCE(e.exam_token,''), ei.invitation_code, e.duration_minutes
		FROM exam_invites ei
		JOIN exams e ON e.id=ei.exam_id AND e.deleted_at IS NULL
		WHERE ei.tenant_id=$1
		  AND ei.deleted_at IS NULL
		  AND ei.student_id=$2
		  AND lower(ei.invitation_code)=lower($3)
		  AND e.status='published'
		LIMIT 1`, tenantID, studentID, code).Scan(&out.ExamID, &out.Name, &out.Description, &out.ExamToken, &out.InvitationCode, &out.DurationMinutes)
	if err != nil {
		return JoinByCodeResponse{}, errors.New("kode undangan tidak valid untuk siswa ini")
	}
	out.Invited = true
	_, _ = s.deps.DB.Exec(ctx, `UPDATE exam_invites SET accepted_at=COALESCE(accepted_at,now()), status='accepted', updated_at=now() WHERE tenant_id=$1 AND exam_id=$2 AND student_id=$3`, tenantID, out.ExamID, studentID)
	return out, nil
}

type StudentExamListResult struct {
	Items []StudentExamView `json:"items"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
	Total int64             `json:"total"`
}

func (s *service) StudentExams(ctx context.Context, tenantID, userID uuid.UUID, q pagination.Query) (StudentExamListResult, error) {
	studentID, err := s.resolveStudentID(ctx, tenantID, userID)
	if err != nil {
		return StudentExamListResult{}, err
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}
	if q.Page < 1 {
		q.Page = 1
	}
	search := "%" + strings.ToLower(strings.TrimSpace(q.Search)) + "%"
	args := []any{tenantID, studentID}
	where := `
		e.tenant_id=$1
		AND e.deleted_at IS NULL
		AND e.status='published'
		AND EXISTS (
			SELECT 1 FROM exam_invites ei
			WHERE ei.tenant_id=e.tenant_id
			  AND ei.exam_id=e.id
			  AND ei.student_id=$2
			  AND ei.deleted_at IS NULL
		)`
	if strings.TrimSpace(q.Search) != "" {
		where += " AND (lower(e.name) LIKE $3 OR lower(e.code) LIKE $3 OR lower(COALESCE(e.description,'')) LIKE $3)"
		args = append(args, search)
	}
	countSQL := "SELECT count(*) FROM exams e WHERE " + where
	var total int64
	if err := s.deps.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return StudentExamListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	rows, err := s.deps.DB.Query(ctx, `
		SELECT e.id, e.code, e.name, COALESCE(e.description,''), e.status,
		       COALESCE(e.exam_token,''), COALESCE(e.duration_minutes,120), COALESCE(e.passing_grade,60),
		       COALESCE(e.max_attempt,1), COALESCE(e.instruction,''), e.metadata,
		       CASE WHEN e.metadata->>'question_count' ~ '^[0-9]+$' THEN (e.metadata->>'question_count')::int ELSE 40 END,
		       COALESCE(ei.invitation_code,''), ei.id IS NOT NULL,
		       es.id, COALESCE(es.status_enum::text,''), es.started_at, es.ends_at, es.submitted_at
		FROM exams e
		JOIN exam_invites ei ON ei.tenant_id=e.tenant_id AND ei.exam_id=e.id AND ei.student_id=$2 AND ei.deleted_at IS NULL
		LEFT JOIN LATERAL (
			SELECT id, status_enum, started_at, ends_at, submitted_at
			FROM exam_sessions
			WHERE tenant_id=e.tenant_id AND exam_id=e.id AND student_id=$2 AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		) es ON true
		WHERE `+where+`
		ORDER BY COALESCE(e.published_at, e.created_at) DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return StudentExamListResult{}, err
	}
	defer rows.Close()
	items := []StudentExamView{}
	for rows.Next() {
		var item StudentExamView
		var raw []byte
		err := rows.Scan(
			&item.ExamID,
			&item.Code,
			&item.Name,
			&item.Description,
			&item.Status,
			&item.ExamToken,
			&item.DurationMinutes,
			&item.PassingGrade,
			&item.MaxAttempt,
			&item.Instruction,
			&raw,
			&item.QuestionCount,
			&item.InvitationCode,
			&item.Invited,
			&item.SessionID,
			&item.SessionStatus,
			&item.StartedAt,
			&item.EndsAt,
			&item.SubmittedAt,
		)
		if err != nil {
			return StudentExamListResult{}, err
		}
		item.StudentID = studentID
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &item.Metadata)
		}
		items = append(items, item)
	}
	return StudentExamListResult{Items: items, Page: q.Page, Limit: q.Limit, Total: total}, rows.Err()
}

func (s *service) resolveStudentID(ctx context.Context, tenantID, userID uuid.UUID) (uuid.UUID, error) {
	var studentID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `SELECT id FROM students WHERE tenant_id=$1 AND user_id=$2 AND deleted_at IS NULL LIMIT 1`, tenantID, userID).Scan(&studentID)
	if err != nil {
		return uuid.Nil, errors.New("student profile not found")
	}
	return studentID, nil
}

type examSettings struct {
	CourseClassID   *uuid.UUID
	DurationMinutes int
	PassingGrade    float64
	ExamToken       string
	RandomQuestion  bool
	RandomOption    bool
	QuestionCount   int
	QuestionPools   []QuestionPoolRequest
	MaxAttempt      int
	Instruction     string
}

func settingsFromCreate(req CreateExamRequest) examSettings {
	return examSettings{
		CourseClassID:   req.CourseClassID,
		DurationMinutes: defaultInt(req.DurationMinutes, 120),
		PassingGrade:    defaultFloat(req.PassingGrade, 60),
		ExamToken:       strings.TrimSpace(req.ExamToken),
		RandomQuestion:  defaultBool(req.RandomQuestion, true),
		RandomOption:    defaultBool(req.RandomOption, true),
		QuestionCount:   defaultInt(req.QuestionCount, 40),
		MaxAttempt:      defaultInt(req.MaxAttempt, 1),
		Instruction:     req.Instruction,
	}
}

func settingsFromUpdate(req UpdateExamRequest) examSettings {
	return examSettings{
		CourseClassID:   req.CourseClassID,
		DurationMinutes: defaultInt(req.DurationMinutes, 120),
		PassingGrade:    defaultFloat(req.PassingGrade, 60),
		ExamToken:       strings.TrimSpace(req.ExamToken),
		RandomQuestion:  defaultBool(req.RandomQuestion, true),
		RandomOption:    defaultBool(req.RandomOption, true),
		QuestionCount:   defaultInt(req.QuestionCount, 40),
		MaxAttempt:      defaultInt(req.MaxAttempt, 1),
		Instruction:     req.Instruction,
	}
}

func settingsFromPublish(req PublishExamRequest, current examSettings) examSettings {
	out := current
	if req.CourseClassID != nil {
		out.CourseClassID = req.CourseClassID
	}
	if req.DurationMinutes > 0 {
		out.DurationMinutes = req.DurationMinutes
	}
	if req.PassingGrade > 0 {
		out.PassingGrade = req.PassingGrade
	}
	if strings.TrimSpace(req.ExamToken) != "" {
		out.ExamToken = strings.TrimSpace(req.ExamToken)
	}
	if req.RandomQuestion != nil {
		out.RandomQuestion = *req.RandomQuestion
	}
	if req.RandomOption != nil {
		out.RandomOption = *req.RandomOption
	}
	if req.QuestionCount > 0 {
		out.QuestionCount = req.QuestionCount
	}
	if len(req.QuestionPools) > 0 {
		out.QuestionPools = normalizeQuestionPools(req.QuestionPools)
	}
	if req.MaxAttempt > 0 {
		out.MaxAttempt = req.MaxAttempt
	}
	if strings.TrimSpace(req.Instruction) != "" {
		out.Instruction = req.Instruction
	}
	out.DurationMinutes = defaultInt(out.DurationMinutes, 120)
	out.PassingGrade = defaultFloat(out.PassingGrade, 60)
	out.QuestionCount = defaultInt(out.QuestionCount, 40)
	out.MaxAttempt = defaultInt(out.MaxAttempt, 1)
	return out
}

func (s *service) currentExamSettings(ctx context.Context, tenantID, examID uuid.UUID) (examSettings, error) {
	out := examSettings{}
	var courseClassID *uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `
		SELECT course_class_id,
		       duration_minutes,
		       passing_grade,
		       COALESCE(exam_token,''),
		       random_question,
		       random_option,
		       CASE WHEN metadata->>'question_count' ~ '^[0-9]+$' THEN (metadata->>'question_count')::int ELSE 40 END,
		       max_attempt,
		       COALESCE(instruction,'')
		FROM exams
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		examID,
		tenantID,
	).Scan(
		&courseClassID,
		&out.DurationMinutes,
		&out.PassingGrade,
		&out.ExamToken,
		&out.RandomQuestion,
		&out.RandomOption,
		&out.QuestionCount,
		&out.MaxAttempt,
		&out.Instruction,
	)
	out.CourseClassID = courseClassID
	if err != nil {
		return out, err
	}
	pools, err := s.currentQuestionPools(ctx, tenantID, examID)
	if err != nil {
		return out, err
	}
	out.QuestionPools = pools
	return out, nil
}

func normalizeQuestionPools(pools []QuestionPoolRequest) []QuestionPoolRequest {
	seen := map[uuid.UUID]int{}
	order := []uuid.UUID{}
	for _, pool := range pools {
		if pool.QuestionTagID == uuid.Nil || pool.QuestionCount <= 0 {
			continue
		}
		if _, ok := seen[pool.QuestionTagID]; !ok {
			order = append(order, pool.QuestionTagID)
		}
		seen[pool.QuestionTagID] += pool.QuestionCount
	}
	out := make([]QuestionPoolRequest, 0, len(order))
	for _, tagID := range order {
		count := seen[tagID]
		if count > 500 {
			count = 500
		}
		out = append(out, QuestionPoolRequest{QuestionTagID: tagID, QuestionCount: count})
	}
	return out
}

func (s *service) validateQuestionPools(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, pools []QuestionPoolRequest) (int, error) {
	total := 0
	tagIDs := []uuid.UUID{}
	admin := canManageAllExamTags(permissions)
	for _, pool := range pools {
		var tagName string
		if err := s.deps.DB.QueryRow(ctx, `
			SELECT name
			FROM question_tags
			WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL
			  AND ($3 OR owner_user_id=$4)`,
			pool.QuestionTagID, tenantID, admin, actorUserID).Scan(&tagName); err != nil {
			return 0, fmt.Errorf("question tag not found: %s", pool.QuestionTagID)
		}
		var available int
		if err := s.deps.DB.QueryRow(ctx, `
			SELECT count(DISTINCT q.id)
			FROM questions q
			JOIN question_tag_relations qtr ON qtr.question_id=q.id AND qtr.deleted_at IS NULL
			WHERE q.tenant_id=$1 AND q.deleted_at IS NULL AND q.status='active' AND qtr.question_tag_id=$2`,
			tenantID, pool.QuestionTagID).Scan(&available); err != nil {
			return 0, err
		}
		if available < pool.QuestionCount {
			return 0, fmt.Errorf("tag %s hanya memiliki %d soal aktif, butuh %d", tagName, available, pool.QuestionCount)
		}
		total += pool.QuestionCount
		tagIDs = append(tagIDs, pool.QuestionTagID)
	}
	if total > 500 {
		return 0, errors.New("total question pool cannot exceed 500")
	}
	var distinctAvailable int
	if err := s.deps.DB.QueryRow(ctx, `
		SELECT count(DISTINCT q.id)
		FROM questions q
		JOIN question_tag_relations qtr ON qtr.question_id=q.id AND qtr.deleted_at IS NULL
		WHERE q.tenant_id=$1 AND q.deleted_at IS NULL AND q.status='active' AND qtr.question_tag_id = ANY($2::uuid[])`,
		tenantID, tagIDs).Scan(&distinctAvailable); err != nil {
		return 0, err
	}
	if distinctAvailable < total {
		return 0, fmt.Errorf("kombinasi tag hanya memiliki %d soal unik aktif, butuh %d", distinctAvailable, total)
	}
	return total, nil
}

func (s *service) replaceQuestionPools(ctx context.Context, tx pgx.Tx, tenantID, examID uuid.UUID, pools []QuestionPoolRequest) error {
	_, err := tx.Exec(ctx, `UPDATE exam_question_pools SET deleted_at=now(), updated_at=now() WHERE tenant_id=$1 AND exam_id=$2 AND deleted_at IS NULL`, tenantID, examID)
	if err != nil {
		return err
	}
	for _, pool := range pools {
		var tagName string
		if err := tx.QueryRow(ctx, `SELECT name FROM question_tags WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, pool.QuestionTagID, tenantID).Scan(&tagName); err != nil {
			return err
		}
		meta, _ := json.Marshal(map[string]any{"question_tag_id": pool.QuestionTagID.String(), "question_count": pool.QuestionCount})
		_, err = tx.Exec(ctx, `
			INSERT INTO exam_question_pools(id,tenant_id,code,name,status,metadata,exam_id,question_tag_id,question_count)
			VALUES($1,$2,$3,$4,'active',$5,$6,$7,$8)`,
			uuid.New(), tenantID, "EQP-"+examID.String()[:8]+"-"+pool.QuestionTagID.String()[:8], tagName, meta, examID, pool.QuestionTagID, pool.QuestionCount)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *service) questionPoolViews(ctx context.Context, tenantID uuid.UUID, pools []QuestionPoolRequest) []QuestionPoolView {
	out := make([]QuestionPoolView, 0, len(pools))
	for _, pool := range pools {
		view := QuestionPoolView{QuestionTagID: pool.QuestionTagID, QuestionCount: pool.QuestionCount}
		_ = s.deps.DB.QueryRow(ctx, `SELECT name FROM question_tags WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, pool.QuestionTagID, tenantID).Scan(&view.QuestionTagName)
		out = append(out, view)
	}
	return out
}

func (s *service) publishBlueprintSnapshot(ctx context.Context, tenantID, examID, actorUserID uuid.UUID, settings examSettings) map[string]any {
	var examCode, examName, examStatus string
	_ = s.deps.DB.QueryRow(ctx, `
		SELECT code, name, status
		FROM exams
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		examID, tenantID).Scan(&examCode, &examName, &examStatus)
	pools := s.questionPoolViews(ctx, tenantID, settings.QuestionPools)
	poolItems := make([]map[string]any, 0, len(pools))
	for _, pool := range pools {
		var available int
		_ = s.deps.DB.QueryRow(ctx, `
			SELECT count(DISTINCT q.id)
			FROM questions q
			JOIN question_tag_relations qtr ON qtr.question_id=q.id AND qtr.deleted_at IS NULL
			WHERE q.tenant_id=$1 AND q.deleted_at IS NULL AND q.status='active' AND qtr.question_tag_id=$2`,
			tenantID, pool.QuestionTagID).Scan(&available)
		poolItems = append(poolItems, map[string]any{
			"question_tag_id":      pool.QuestionTagID.String(),
			"question_tag_name":    pool.QuestionTagName,
			"question_count":       pool.QuestionCount,
			"available_at_publish": available,
		})
	}
	out := map[string]any{
		"exam_id":          examID.String(),
		"exam_code":        examCode,
		"exam_name":        examName,
		"previous_status":  examStatus,
		"published_by":     actorUserID.String(),
		"captured_at":      time.Now().UTC().Format(time.RFC3339Nano),
		"duration_minutes": settings.DurationMinutes,
		"passing_grade":    settings.PassingGrade,
		"exam_token":       settings.ExamToken,
		"random_question":  settings.RandomQuestion,
		"random_option":    settings.RandomOption,
		"question_count":   settings.QuestionCount,
		"max_attempt":      settings.MaxAttempt,
		"instruction":      settings.Instruction,
		"question_pools":   poolItems,
	}
	if settings.CourseClassID != nil {
		out["course_class_id"] = settings.CourseClassID.String()
	}
	return out
}

func insertExamAudit(ctx context.Context, tx pgx.Tx, tenantID, actorUserID uuid.UUID, eventType string, entityID uuid.UUID, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["actor_user_id"] = actorUserID.String()
	metadata["entity_id"] = entityID.String()
	metadata["entity_type"] = "exam"
	metadata["event_type"] = eventType
	metadata["recorded_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	raw, _ := json.Marshal(metadata)
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs(id,tenant_id,code,name,description,status,metadata)
		VALUES($1,$2,$3,$4,$5,'active',$6)`,
		uuid.New(), tenantID, "AUD-"+entityID.String()[:8], eventType, "Academic audit trail", raw)
	return err
}

func (s *service) currentQuestionPools(ctx context.Context, tenantID, examID uuid.UUID) ([]QuestionPoolRequest, error) {
	rows, err := s.deps.DB.Query(ctx, `
		SELECT question_tag_id, question_count
		FROM exam_question_pools
		WHERE tenant_id=$1
		  AND exam_id=$2
		  AND deleted_at IS NULL
		  AND question_tag_id IS NOT NULL
		  AND question_count > 0
		ORDER BY created_at ASC`, tenantID, examID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QuestionPoolRequest{}
	for rows.Next() {
		var item QuestionPoolRequest
		if err := rows.Scan(&item.QuestionTagID, &item.QuestionCount); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) applyExamSettings(ctx context.Context, tenantID, examID, ownerUserID uuid.UUID, settings examSettings, status string) error {
	_, err := s.deps.DB.Exec(ctx, `
		UPDATE exams
		SET owner_user_id=CASE WHEN $1='00000000-0000-0000-0000-000000000000'::uuid THEN owner_user_id ELSE $1 END,
		    exam_token=$2,
		    course_class_id=$3,
		    duration_minutes=$4,
		    passing_grade=$5,
		    random_question=$6,
		    random_option=$7,
		    max_attempt=$8,
		    instruction=$9,
		    published_at=CASE WHEN $10='published' THEN COALESCE(published_at, now()) ELSE published_at END,
		    updated_at=now()
		WHERE id=$11 AND tenant_id=$12 AND deleted_at IS NULL`,
		ownerUserID,
		settings.ExamToken,
		settings.CourseClassID,
		settings.DurationMinutes,
		settings.PassingGrade,
		settings.RandomQuestion,
		settings.RandomOption,
		settings.MaxAttempt,
		settings.Instruction,
		status,
		examID,
		tenantID,
	)
	return err
}

func mergeExamMetadata(metadata map[string]any, settings examSettings) {
	for key, value := range examSettingsMetadata(settings) {
		metadata[key] = value
	}
}

func examSettingsMetadata(settings examSettings) map[string]any {
	out := map[string]any{
		"exam_token":       settings.ExamToken,
		"duration_minutes": settings.DurationMinutes,
		"passing_grade":    settings.PassingGrade,
		"random_question":  settings.RandomQuestion,
		"random_option":    settings.RandomOption,
		"question_count":   settings.QuestionCount,
		"max_attempt":      settings.MaxAttempt,
		"instruction":      settings.Instruction,
	}
	if settings.CourseClassID != nil {
		out["course_class_id"] = settings.CourseClassID.String()
	}
	if len(settings.QuestionPools) > 0 {
		pools := make([]map[string]any, 0, len(settings.QuestionPools))
		for _, pool := range settings.QuestionPools {
			pools = append(pools, map[string]any{
				"question_tag_id": pool.QuestionTagID.String(),
				"question_count":  pool.QuestionCount,
			})
		}
		out["question_pools"] = pools
	}
	return out
}

func mustJSON(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func defaultInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func defaultFloat(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func defaultBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func (s *service) ensureExamWritable(ctx context.Context, tenantID, examID, userID uuid.UUID, permissions []string) error {
	var owner uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `SELECT COALESCE(owner_user_id,'00000000-0000-0000-0000-000000000000') FROM exams WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, examID, tenantID).Scan(&owner)
	if err != nil {
		return err
	}
	if canManageAllExams(permissions) {
		return nil
	}
	if owner != uuid.Nil && owner != userID {
		return errors.New("exam belongs to another lecturer")
	}
	return nil
}

func canManageAllExams(permissions []string) bool {
	for _, permission := range permissions {
		if permission == "*" || permission == "users:read" || permission == "tenants:read" {
			return true
		}
	}
	return false
}

func canManageAllExamTags(permissions []string) bool {
	for _, permission := range permissions {
		if permission == "*" || permission == "users:read" || permission == "tenants:read" {
			return true
		}
	}
	return false
}

func newCode(prefix string, n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, n)
	for i := range out {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			out[i] = alphabet[i%len(alphabet)]
			continue
		}
		out[i] = alphabet[idx.Int64()]
	}
	return prefix + "-" + string(out)
}
func (s *service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	var status string
	if err := s.deps.DB.QueryRow(ctx, `SELECT status FROM exams WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, id, tenantID).Scan(&status); err != nil {
		return err
	}
	if status != "draft" {
		return errors.New("only draft exams can be deleted")
	}
	err := s.repo.Delete(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "exams.deleted", []byte(id.String()))
	}
	return err
}
