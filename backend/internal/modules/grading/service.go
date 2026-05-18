package grading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
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
	Create(context.Context, uuid.UUID, CreateGradingRequest) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateGradingRequest) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	SessionGrades(context.Context, uuid.UUID, uuid.UUID) (SessionGradesResponse, error)
	ManualScore(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string, ManualScoreRequest) (SessionGradesResponse, error)
	ReviewSessions(context.Context, uuid.UUID, uuid.UUID, []string, pagination.Query, uuid.UUID) (ReviewSessionListResult, error)
	ReviewSessionDetail(context.Context, uuid.UUID, uuid.UUID, []string, uuid.UUID) (ReviewSessionDetailResponse, error)
	ReleaseSessionResult(context.Context, uuid.UUID, uuid.UUID, []string, uuid.UUID) (ReviewSessionDetailResponse, error)
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
func (s *service) Create(ctx context.Context, tenantID uuid.UUID, req CreateGradingRequest) (shared.Record, error) {
	if err := ValidateCreate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Create(ctx, tenantID, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "grading.created", body)
	}
	return out, err
}
func (s *service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateGradingRequest) (shared.Record, error) {
	if err := ValidateUpdate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Update(ctx, tenantID, id, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "grading.updated", body)
	}
	return out, err
}
func (s *service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "grading.deleted", []byte(id.String()))
	}
	return err
}

func (s *service) SessionGrades(ctx context.Context, tenantID, sessionID uuid.UUID) (SessionGradesResponse, error) {
	return s.sessionGrades(ctx, s.deps.DB, tenantID, sessionID)
}

func (s *service) ManualScore(ctx context.Context, tenantID, gradingID, reviewerUserID uuid.UUID, permissions []string, req ManualScoreRequest) (SessionGradesResponse, error) {
	if err := validate.Struct(req); err != nil {
		return SessionGradesResponse{}, err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return SessionGradesResponse{}, err
	}
	defer tx.Rollback(ctx)

	var raw []byte
	if err := tx.QueryRow(ctx, `
		SELECT metadata
		FROM grading
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL
		FOR UPDATE`, gradingID, tenantID).Scan(&raw); err != nil {
		return SessionGradesResponse{}, err
	}
	meta := map[string]any{}
	_ = json.Unmarshal(raw, &meta)
	sessionID, err := uuid.Parse(stringFromAny(meta["exam_session_id"]))
	if err != nil || sessionID == uuid.Nil {
		return SessionGradesResponse{}, errors.New("grading row has no exam session id")
	}
	if err := s.ensureReviewAccess(ctx, tx, tenantID, reviewerUserID, permissions, sessionID); err != nil {
		return SessionGradesResponse{}, err
	}
	maxScore := numberFromAny(meta["max_score"])
	if maxScore <= 0 {
		maxScore = req.EarnedScore
	}
	if req.EarnedScore > maxScore {
		return SessionGradesResponse{}, errors.New("earned_score cannot exceed max_score")
	}
	status := req.Status
	if status == "" {
		status = "reviewed"
	}
	meta["earned_score"] = req.EarnedScore
	meta["manual_score"] = req.EarnedScore
	meta["manual_status"] = status
	meta["feedback"] = req.Feedback
	meta["reviewer_user_id"] = reviewerUserID.String()
	meta["reviewed_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	meta["is_correct"] = maxScore > 0 && req.EarnedScore >= maxScore
	meta["answered"] = true
	next, _ := json.Marshal(meta)
	if _, err := tx.Exec(ctx, `
		UPDATE grading
		SET metadata=$1, status=$2, updated_at=now()
		WHERE id=$3 AND tenant_id=$4 AND deleted_at IS NULL`, next, status, gradingID, tenantID); err != nil {
		return SessionGradesResponse{}, err
	}
	if err := s.recalculateSessionSummary(ctx, tx, tenantID, sessionID); err != nil {
		return SessionGradesResponse{}, err
	}
	out, err := s.sessionGrades(ctx, tx, tenantID, sessionID)
	if err != nil {
		return SessionGradesResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SessionGradesResponse{}, err
	}
	if s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "grading.manual.reviewed", body)
	}
	return out, nil
}

type gradeQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *service) sessionGrades(ctx context.Context, q gradeQueryer, tenantID, sessionID uuid.UUID) (SessionGradesResponse, error) {
	out := SessionGradesResponse{ExamSessionID: sessionID, Items: []GradeItemView{}}
	var rawSession []byte
	if err := q.QueryRow(ctx, `
		SELECT status_enum::text, COALESCE(metadata,'{}'::jsonb)
		FROM exam_sessions
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, sessionID, tenantID).Scan(&out.Status, &rawSession); err != nil {
		return out, err
	}
	sessionMeta := map[string]any{}
	_ = json.Unmarshal(rawSession, &sessionMeta)
	out.Score = numberFromAny(sessionMeta["score"])
	out.MaxScore = numberFromAny(sessionMeta["max_score"])
	out.Percentage = numberFromAny(sessionMeta["percentage"])
	out.Passed = boolFromAny(sessionMeta["passed"])

	rows, err := q.Query(ctx, `
		SELECT id, metadata
		FROM grading
		WHERE tenant_id=$1 AND deleted_at IS NULL AND metadata->>'exam_session_id'=$2
		ORDER BY code ASC, created_at ASC`, tenantID, sessionID.String())
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item GradeItemView
		var raw []byte
		if err := rows.Scan(&item.ID, &raw); err != nil {
			return out, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(raw, &item.Metadata)
		item.ExamSessionID = sessionID
		item.SessionQuestionID = uuidFromMeta(item.Metadata, "session_question_id")
		item.QuestionID = uuidFromMeta(item.Metadata, "question_id")
		item.AnswerMode = stringFromAny(item.Metadata["answer_mode"])
		item.SelectedOptionIDs = stringSliceFromAny(item.Metadata["selected_option_ids"])
		item.CorrectOptionIDs = stringSliceFromAny(item.Metadata["correct_option_ids"])
		item.EarnedScore = numberFromAny(item.Metadata["earned_score"])
		item.MaxScore = numberFromAny(item.Metadata["max_score"])
		item.Answered = boolFromAny(item.Metadata["answered"])
		item.IsCorrect = boolFromAny(item.Metadata["is_correct"])
		item.ManualStatus = stringFromAny(item.Metadata["manual_status"])
		item.Feedback = stringFromAny(item.Metadata["feedback"])
		item.ManualRequired = item.AnswerMode != "single" && item.AnswerMode != "multiple"
		if snapshot := mapFromAny(item.Metadata["question_snapshot"]); len(snapshot) > 0 {
			item.QuestionText = stringFromAny(snapshot["text"])
		}
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}

func (s *service) ReviewSessions(ctx context.Context, tenantID, reviewerUserID uuid.UUID, permissions []string, q pagination.Query, examID uuid.UUID) (ReviewSessionListResult, error) {
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}
	if q.Page < 1 {
		q.Page = 1
	}
	args := []any{tenantID}
	where := "es.tenant_id=$1 AND es.deleted_at IS NULL AND es.status_enum='completed'"
	if examID != uuid.Nil {
		args = append(args, examID)
		where += fmt.Sprintf(" AND es.exam_id=$%d", len(args))
	}
	if !canReviewAll(permissions) {
		args = append(args, reviewerUserID)
		where += fmt.Sprintf(" AND e.owner_user_id=$%d", len(args))
	}
	if search := strings.TrimSpace(q.Search); search != "" {
		args = append(args, "%"+strings.ToLower(search)+"%")
		where += fmt.Sprintf(" AND (lower(e.name) LIKE $%d OR lower(s.name) LIKE $%d OR lower(s.code) LIKE $%d)", len(args), len(args), len(args))
	}
	var total int64
	if err := s.deps.DB.QueryRow(ctx, `
		SELECT count(*)
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id
		JOIN students s ON s.id=es.student_id
		WHERE `+where, args...).Scan(&total); err != nil {
		return ReviewSessionListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	rows, err := s.deps.DB.Query(ctx, `
		SELECT es.id, es.exam_id, e.name, es.student_id, s.code, s.name, es.status_enum::text,
		       es.started_at, es.submitted_at, COALESCE(es.metadata,'{}'::jsonb),
		       COALESCE((
		         SELECT count(*) FROM grading g
		         WHERE g.tenant_id=es.tenant_id AND g.deleted_at IS NULL
		           AND g.metadata->>'exam_session_id'=es.id::text
		           AND COALESCE(g.metadata->>'manual_status','') IN ('pending','needs_review')
		       ),0)
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id
		JOIN students s ON s.id=es.student_id
		WHERE `+where+`
		ORDER BY es.submitted_at DESC NULLS LAST, es.created_at DESC
		LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return ReviewSessionListResult{}, err
	}
	defer rows.Close()
	out := ReviewSessionListResult{Items: []ReviewSessionView{}, Page: q.Page, Limit: q.Limit, Total: total}
	for rows.Next() {
		var item ReviewSessionView
		var raw []byte
		var startedAt, submittedAt *time.Time
		if err := rows.Scan(&item.SessionID, &item.ExamID, &item.ExamName, &item.StudentID, &item.StudentCode, &item.StudentName, &item.Status, &startedAt, &submittedAt, &raw, &item.ManualRequired); err != nil {
			return out, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(raw, &item.Metadata)
		item.Score = numberFromAny(item.Metadata["score"])
		item.MaxScore = numberFromAny(item.Metadata["max_score"])
		item.Percentage = numberFromAny(item.Metadata["percentage"])
		item.Passed = boolFromAny(item.Metadata["passed"])
		if startedAt != nil {
			item.StartedAt = startedAt.UTC().Format(time.RFC3339)
		}
		if submittedAt != nil {
			item.SubmittedAt = submittedAt.UTC().Format(time.RFC3339)
		}
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}

func (s *service) ReviewSessionDetail(ctx context.Context, tenantID, reviewerUserID uuid.UUID, permissions []string, sessionID uuid.UUID) (ReviewSessionDetailResponse, error) {
	if err := s.ensureReviewAccess(ctx, s.deps.DB, tenantID, reviewerUserID, permissions, sessionID); err != nil {
		return ReviewSessionDetailResponse{}, err
	}
	list, err := s.ReviewSessions(ctx, tenantID, reviewerUserID, permissions, pagination.Query{Page: 1, Limit: 1}, uuid.Nil)
	if err != nil {
		return ReviewSessionDetailResponse{}, err
	}
	var session ReviewSessionView
	for _, item := range list.Items {
		if item.SessionID == sessionID {
			session = item
			break
		}
	}
	if session.SessionID == uuid.Nil {
		var rawSession []byte
		if err := s.deps.DB.QueryRow(ctx, `
			SELECT es.id, es.exam_id, e.name, es.student_id, s.code, s.name, es.status_enum::text, COALESCE(es.metadata,'{}'::jsonb)
			FROM exam_sessions es JOIN exams e ON e.id=es.exam_id JOIN students s ON s.id=es.student_id
			WHERE es.id=$1 AND es.tenant_id=$2 AND es.deleted_at IS NULL`, sessionID, tenantID).
			Scan(&session.SessionID, &session.ExamID, &session.ExamName, &session.StudentID, &session.StudentCode, &session.StudentName, &session.Status, &rawSession); err != nil {
			return ReviewSessionDetailResponse{}, err
		}
		session.Metadata = map[string]any{}
		_ = json.Unmarshal(rawSession, &session.Metadata)
		session.Score = numberFromAny(session.Metadata["score"])
		session.MaxScore = numberFromAny(session.Metadata["max_score"])
		session.Percentage = numberFromAny(session.Metadata["percentage"])
		session.Passed = boolFromAny(session.Metadata["passed"])
	}
	grades, err := s.reviewGradesBySessionQuestion(ctx, tenantID, sessionID)
	if err != nil {
		return ReviewSessionDetailResponse{}, err
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT esq.id, esq.question_id, esq.position, COALESCE(esq.metadata,'{}'::jsonb), COALESCE(a.answer_payload,'{}'::jsonb)
		FROM exam_session_questions esq
		LEFT JOIN answers a ON a.exam_session_id=esq.exam_session_id AND a.question_id=esq.question_id AND a.deleted_at IS NULL
		WHERE esq.tenant_id=$1 AND esq.exam_session_id=$2 AND esq.deleted_at IS NULL
		ORDER BY esq.position ASC`, tenantID, sessionID)
	if err != nil {
		return ReviewSessionDetailResponse{}, err
	}
	defer rows.Close()
	out := ReviewSessionDetailResponse{Session: session, Items: []ReviewQuestionView{}}
	for rows.Next() {
		var item ReviewQuestionView
		var rawMeta, rawAnswer []byte
		if err := rows.Scan(&item.SessionQuestionID, &item.QuestionID, &item.Position, &rawMeta, &rawAnswer); err != nil {
			return out, err
		}
		item.AnswerPayload = map[string]any{}
		_ = json.Unmarshal(rawAnswer, &item.AnswerPayload)
		item.QuestionTagName = reviewStringFromMetadata(rawMeta, "question_tag_name")
		snapshot := reviewSnapshot(rawMeta)
		item.Text = stringFromAny(snapshot["text"])
		item.AnswerMode = stringFromAny(snapshot["answer_mode"])
		item.MaxScore = numberFromAny(snapshot["score"])
		item.Media = reviewMediaList(snapshot["media"])
		item.Options = reviewOptions(snapshot["options"])
		if grade, ok := grades[item.SessionQuestionID]; ok {
			item.GradingID = grade.ID
			item.SelectedOptionIDs = grade.SelectedOptionIDs
			item.CorrectOptionIDs = grade.CorrectOptionIDs
			item.EarnedScore = grade.EarnedScore
			item.MaxScore = grade.MaxScore
			item.Answered = grade.Answered
			item.IsCorrect = grade.IsCorrect
			item.ManualRequired = grade.ManualRequired
			item.ManualStatus = grade.ManualStatus
			item.Feedback = grade.Feedback
		}
		selected := map[string]bool{}
		correct := map[string]bool{}
		for _, id := range item.SelectedOptionIDs {
			selected[id] = true
		}
		for _, id := range item.CorrectOptionIDs {
			correct[id] = true
		}
		for i := range item.Options {
			item.Options[i].Selected = selected[item.Options[i].ID]
			item.Options[i].Correct = correct[item.Options[i].ID]
		}
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}

func (s *service) ReleaseSessionResult(ctx context.Context, tenantID, reviewerUserID uuid.UUID, permissions []string, sessionID uuid.UUID) (ReviewSessionDetailResponse, error) {
	if err := s.ensureReviewAccess(ctx, s.deps.DB, tenantID, reviewerUserID, permissions, sessionID); err != nil {
		return ReviewSessionDetailResponse{}, err
	}
	release := map[string]any{
		"result_policy": map[string]any{
			"visibility":           "manual_release",
			"released":             true,
			"released_at":          time.Now().UTC().Format(time.RFC3339Nano),
			"released_by":          reviewerUserID.String(),
			"show_question_detail": false,
		},
	}
	raw, _ := json.Marshal(release)
	if _, err := s.deps.DB.Exec(ctx, `
		UPDATE exam_sessions
		SET metadata=COALESCE(metadata,'{}'::jsonb) || $1::jsonb, updated_at=now()
		WHERE id=$2 AND tenant_id=$3 AND deleted_at IS NULL`, raw, sessionID, tenantID); err != nil {
		return ReviewSessionDetailResponse{}, err
	}
	return s.ReviewSessionDetail(ctx, tenantID, reviewerUserID, permissions, sessionID)
}

func (s *service) ensureReviewAccess(ctx context.Context, q gradeQueryer, tenantID, userID uuid.UUID, permissions []string, sessionID uuid.UUID) error {
	if canReviewAll(permissions) {
		var exists bool
		if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM exam_sessions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)`, sessionID, tenantID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return errors.New("exam session not found")
		}
		return nil
	}
	var owner uuid.UUID
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(e.owner_user_id,'00000000-0000-0000-0000-000000000000')
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
		WHERE es.id=$1 AND es.tenant_id=$2 AND es.deleted_at IS NULL`, sessionID, tenantID).Scan(&owner); err != nil {
		return err
	}
	if owner != userID {
		return errors.New("exam result belongs to another lecturer")
	}
	return nil
}

func canReviewAll(permissions []string) bool {
	for _, permission := range permissions {
		if permission == "*" || permission == "users:read" || permission == "tenants:read" || permission == "grading:review_all" {
			return true
		}
	}
	return false
}

func (s *service) reviewGradesBySessionQuestion(ctx context.Context, tenantID, sessionID uuid.UUID) (map[uuid.UUID]GradeItemView, error) {
	grades, err := s.sessionGrades(ctx, s.deps.DB, tenantID, sessionID)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]GradeItemView{}
	for _, item := range grades.Items {
		out[item.SessionQuestionID] = item
	}
	return out, nil
}

func reviewStringFromMetadata(raw []byte, key string) string {
	meta := map[string]any{}
	_ = json.Unmarshal(raw, &meta)
	return stringFromAny(meta[key])
}

func reviewSnapshot(raw []byte) map[string]any {
	meta := map[string]any{}
	_ = json.Unmarshal(raw, &meta)
	return mapFromAny(meta["snapshot"])
}

func reviewMediaList(value any) []ReviewMediaView {
	values, ok := value.([]any)
	if !ok {
		return []ReviewMediaView{}
	}
	out := []ReviewMediaView{}
	for _, value := range values {
		item := mapFromAny(value)
		id := stringFromAny(item["id"])
		if id == "" {
			continue
		}
		out = append(out, ReviewMediaView{
			ID:        id,
			URL:       "/questions/media/" + id + "/content",
			MimeType:  stringFromAny(item["mime_type"]),
			FileSize:  int64(numberFromAny(item["file_size"])),
			SortOrder: int(numberFromAny(item["sort_order"])),
		})
	}
	return out
}

func reviewOptions(value any) []ReviewOptionView {
	values, ok := value.([]any)
	if !ok {
		return []ReviewOptionView{}
	}
	out := []ReviewOptionView{}
	for _, value := range values {
		item := mapFromAny(value)
		out = append(out, ReviewOptionView{
			ID:    stringFromAny(item["id"]),
			Label: stringFromAny(item["label"]),
			Text:  stringFromAny(item["text"]),
			Media: reviewMediaList(item["media"]),
		})
	}
	return out
}

func (s *service) recalculateSessionSummary(ctx context.Context, tx pgx.Tx, tenantID, sessionID uuid.UUID) error {
	rows, err := tx.Query(ctx, `
		SELECT metadata
		FROM grading
		WHERE tenant_id=$1 AND deleted_at IS NULL AND metadata->>'exam_session_id'=$2`, tenantID, sessionID.String())
	if err != nil {
		return err
	}
	score := 0.0
	maxScore := 0.0
	correct := 0
	wrong := 0
	answered := 0
	unanswered := 0
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		meta := map[string]any{}
		_ = json.Unmarshal(raw, &meta)
		earned := numberFromAny(meta["earned_score"])
		max := numberFromAny(meta["max_score"])
		isAnswered := boolFromAny(meta["answered"])
		isCorrect := boolFromAny(meta["is_correct"])
		score += earned
		maxScore += max
		if isAnswered {
			answered++
			if isCorrect {
				correct++
			} else {
				wrong++
			}
		} else {
			unanswered++
		}
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return rowsErr
	}
	var passingGrade float64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(e.passing_grade,60)
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
		WHERE es.id=$1 AND es.tenant_id=$2 AND es.deleted_at IS NULL`, sessionID, tenantID).Scan(&passingGrade); err != nil {
		return err
	}
	percentage := 0.0
	if maxScore > 0 {
		percentage = math.Round((score/maxScore)*10000) / 100
	}
	meta, _ := json.Marshal(map[string]any{
		"score":                 score,
		"max_score":             maxScore,
		"percentage":            percentage,
		"passing_grade":         passingGrade,
		"passed":                percentage >= passingGrade,
		"correct_count":         correct,
		"wrong_count":           wrong,
		"answered_count":        answered,
		"unanswered_count":      unanswered,
		"manual_grading_synced": time.Now().UTC().Format(time.RFC3339Nano),
	})
	_, err = tx.Exec(ctx, `
		UPDATE exam_sessions
		SET metadata=COALESCE(metadata,'{}'::jsonb) || $1::jsonb, updated_at=now()
		WHERE id=$2 AND tenant_id=$3 AND deleted_at IS NULL`, meta, sessionID, tenantID)
	return err
}

func uuidFromMeta(meta map[string]any, key string) uuid.UUID {
	id, _ := uuid.Parse(stringFromAny(meta[key]))
	return id
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return []string{}
	}
}

func mapFromAny(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func numberFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case json.Number:
		out, _ := typed.Float64()
		return out
	default:
		return 0
	}
}

func boolFromAny(value any) bool {
	typed, _ := value.(bool)
	return typed
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return fmt.Sprint(value)
}
