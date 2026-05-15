package exam_sessions

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
)

type Service interface {
	List(context.Context, uuid.UUID, pagination.Query) (shared.ListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (shared.Record, error)
	Create(context.Context, uuid.UUID, CreateExamSessionRequest) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateExamSessionRequest) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	StudentHistory(context.Context, uuid.UUID, uuid.UUID, pagination.Query) (StudentSessionListResult, error)
	StudentResultDetail(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (StudentResultDetailResponse, error)
	SessionQuestions(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (ExamQuestionsResponse, error)
	SubmitExam(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, SubmitExamRequest) (SubmitExamResponse, error)
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
func (s *service) Create(ctx context.Context, tenantID uuid.UUID, req CreateExamSessionRequest) (shared.Record, error) {
	if err := ValidateCreate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Create(ctx, tenantID, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "exam_sessions.created", body)
	}
	return out, err
}
func (s *service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateExamSessionRequest) (shared.Record, error) {
	if err := ValidateUpdate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Update(ctx, tenantID, id, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "exam_sessions.updated", body)
	}
	return out, err
}
func (s *service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "exam_sessions.deleted", []byte(id.String()))
	}
	return err
}

func (s *service) StudentHistory(ctx context.Context, tenantID, userID uuid.UUID, q pagination.Query) (StudentSessionListResult, error) {
	studentID, err := s.resolveStudentID(ctx, tenantID, userID)
	if err != nil {
		return StudentSessionListResult{}, err
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}
	if q.Page < 1 {
		q.Page = 1
	}
	search := "%" + strings.ToLower(strings.TrimSpace(q.Search)) + "%"
	args := []any{tenantID, studentID}
	where := "es.tenant_id=$1 AND es.student_id=$2 AND es.deleted_at IS NULL"
	if strings.TrimSpace(q.Search) != "" {
		where += " AND (lower(e.name) LIKE $3 OR lower(es.code) LIKE $3)"
		args = append(args, search)
	}
	var total int64
	if err := s.deps.DB.QueryRow(ctx, `SELECT count(*) FROM exam_sessions es JOIN exams e ON e.id=es.exam_id WHERE `+where, args...).Scan(&total); err != nil {
		return StudentSessionListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	rows, err := s.deps.DB.Query(ctx, `
		SELECT es.id, es.exam_id, es.student_id, es.code, e.name, es.status_enum::text,
		       es.started_at, es.ends_at, es.submitted_at, es.metadata,
		       NULLIF(es.metadata->>'score','')::numeric
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id
		WHERE `+where+`
		ORDER BY es.created_at DESC
		LIMIT $`+stringIndex(len(args)-1)+` OFFSET $`+stringIndex(len(args)),
		args...,
	)
	if err != nil {
		return StudentSessionListResult{}, err
	}
	defer rows.Close()
	items := []StudentSessionView{}
	now := time.Now().UTC()
	for rows.Next() {
		var item StudentSessionView
		var raw []byte
		err := rows.Scan(&item.SessionID, &item.ExamID, &item.StudentID, &item.Code, &item.ExamName, &item.Status, &item.StartedAt, &item.EndsAt, &item.SubmittedAt, &raw, &item.Score)
		if err != nil {
			return StudentSessionListResult{}, err
		}
		if item.EndsAt != nil {
			item.RemainingSeconds = int64(math.Max(0, item.EndsAt.Sub(now).Seconds()))
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &item.Metadata)
		}
		recovery := mapFromAny(item.Metadata["recovery"])
		if item.Status == "reconnecting" && boolFromAny(recovery["timer_paused"]) {
			item.RemainingSeconds = int64(math.Max(0, numberFromAny(recovery["remaining_seconds"])))
		}
		items = append(items, item)
	}
	return StudentSessionListResult{Items: items, Page: q.Page, Limit: q.Limit, Total: total}, rows.Err()
}

func (s *service) StudentResultDetail(ctx context.Context, tenantID, userID, sessionID uuid.UUID) (StudentResultDetailResponse, error) {
	studentID, err := s.resolveStudentID(ctx, tenantID, userID)
	if err != nil {
		return StudentResultDetailResponse{}, err
	}
	state, err := s.autoSubmitIfExpired(ctx, tenantID, sessionID)
	if err != nil {
		return StudentResultDetailResponse{}, err
	}
	if state.StudentID != studentID {
		return StudentResultDetailResponse{}, errors.New("exam session belongs to another student")
	}

	var out StudentResultDetailResponse
	var rawSession []byte
	err = s.deps.DB.QueryRow(ctx, `
		SELECT es.id, es.exam_id, es.student_id, es.code, e.name, es.status_enum::text,
		       es.started_at, es.ends_at, es.submitted_at, COALESCE(es.metadata,'{}'::jsonb),
		       NULLIF(es.metadata->>'score','')::numeric
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id
		WHERE es.id=$1 AND es.tenant_id=$2 AND es.student_id=$3 AND es.deleted_at IS NULL`,
		sessionID, tenantID, studentID,
	).Scan(
		&out.Session.SessionID,
		&out.Session.ExamID,
		&out.Session.StudentID,
		&out.Session.Code,
		&out.Session.ExamName,
		&out.Session.Status,
		&out.Session.StartedAt,
		&out.Session.EndsAt,
		&out.Session.SubmittedAt,
		&rawSession,
		&out.Session.Score,
	)
	if err != nil {
		return StudentResultDetailResponse{}, err
	}
	out.Session.RemainingSeconds = state.RemainingSeconds
	out.Session.Metadata = map[string]any{}
	_ = json.Unmarshal(rawSession, &out.Session.Metadata)
	out.Summary = map[string]any{
		"score":            numberFromMeta(out.Session.Metadata, "score"),
		"max_score":        numberFromMeta(out.Session.Metadata, "max_score"),
		"percentage":       numberFromMeta(out.Session.Metadata, "percentage"),
		"passing_grade":    numberFromMeta(out.Session.Metadata, "passing_grade"),
		"passed":           boolFromAny(out.Session.Metadata["passed"]),
		"correct_count":    int(numberFromMeta(out.Session.Metadata, "correct_count")),
		"wrong_count":      int(numberFromMeta(out.Session.Metadata, "wrong_count")),
		"answered_count":   int(numberFromMeta(out.Session.Metadata, "answered_count")),
		"unanswered_count": int(numberFromMeta(out.Session.Metadata, "unanswered_count")),
	}

	grades, err := s.resultGradeMetadata(ctx, tenantID, sessionID)
	if err != nil {
		return StudentResultDetailResponse{}, err
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT esq.id, esq.question_id, esq.position, esq.code,
		       COALESCE(q.content, q.name, ''), COALESCE(q.question_type, 'multiple_choice'), COALESCE(q.answer_mode, 'single'),
		       COALESCE(esq.metadata, '{}'::jsonb), COALESCE(a.answer_payload, '{}'::jsonb)
		FROM exam_session_questions esq
		LEFT JOIN questions q ON q.id=esq.question_id
		LEFT JOIN answers a ON a.exam_session_id=esq.exam_session_id AND a.question_id=esq.question_id AND a.deleted_at IS NULL
		WHERE esq.tenant_id=$1 AND esq.exam_session_id=$2 AND esq.deleted_at IS NULL
		ORDER BY esq.position ASC`, tenantID, sessionID)
	if err != nil {
		return StudentResultDetailResponse{}, err
	}
	defer rows.Close()
	out.Questions = []StudentResultQuestionView{}
	for rows.Next() {
		var item StudentResultQuestionView
		var rawQuestion, rawAnswer []byte
		if err := rows.Scan(&item.SessionQuestionID, &item.QuestionID, &item.Position, &item.Code, &item.Text, &item.QuestionType, &item.AnswerMode, &rawQuestion, &rawAnswer); err != nil {
			return StudentResultDetailResponse{}, err
		}
		item.AnswerPayload = map[string]any{}
		_ = json.Unmarshal(rawAnswer, &item.AnswerPayload)
		if snapshot, ok := snapshotFromMetadata(rawQuestion); ok {
			item.QuestionID = snapshot.QuestionID
			item.Text = snapshot.Text
			item.QuestionType = snapshot.QuestionType
			item.AnswerMode = snapshot.AnswerMode
			item.Media = snapshot.Media.withURLs()
			item.Options = snapshot.examOptions()
			item.CorrectOptionIDs = uuidStrings(snapshot.correctOptionIDs())
			item.MaxScore = snapshot.Score
		}
		item.SelectedOptionIDs = uuidStrings(uuidSliceFromAny(item.AnswerPayload["selected_option_ids"]))
		item.Answered = len(item.SelectedOptionIDs) > 0 || strings.TrimSpace(stringFromAny(item.AnswerPayload["text"], "")) != ""
		if gradeMeta, ok := grades[item.SessionQuestionID]; ok {
			item.Metadata = gradeMeta
			item.SelectedOptionIDs = resultStringSliceFromAny(gradeMeta["selected_option_ids"])
			item.CorrectOptionIDs = resultStringSliceFromAny(gradeMeta["correct_option_ids"])
			item.EarnedScore = numberFromAny(gradeMeta["earned_score"])
			item.MaxScore = numberFromAny(gradeMeta["max_score"])
			item.Answered = boolFromAny(gradeMeta["answered"])
			item.IsCorrect = boolFromAny(gradeMeta["is_correct"])
			item.ManualStatus = stringFromAny(gradeMeta["manual_status"], "")
			item.Feedback = stringFromAny(gradeMeta["feedback"], "")
		}
		if item.MaxScore <= 0 {
			item.MaxScore = 1
		}
		item.ManualRequired = item.AnswerMode != "single" && item.AnswerMode != "multiple"
		out.Questions = append(out.Questions, item)
	}
	return out, rows.Err()
}

func (s *service) resultGradeMetadata(ctx context.Context, tenantID, sessionID uuid.UUID) (map[uuid.UUID]map[string]any, error) {
	rows, err := s.deps.DB.Query(ctx, `
		SELECT metadata
		FROM grading
		WHERE tenant_id=$1 AND deleted_at IS NULL AND metadata->>'exam_session_id'=$2
		ORDER BY created_at ASC`, tenantID, sessionID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]map[string]any{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		meta := map[string]any{}
		_ = json.Unmarshal(raw, &meta)
		sessionQuestionID, err := uuid.Parse(stringFromAny(meta["session_question_id"], ""))
		if err == nil && sessionQuestionID != uuid.Nil {
			out[sessionQuestionID] = meta
		}
	}
	return out, rows.Err()
}

func resultStringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := stringFromAny(item, "")
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		if typed == "" {
			return []string{}
		}
		return []string{typed}
	default:
		return []string{}
	}
}

func (s *service) resolveStudentID(ctx context.Context, tenantID, userID uuid.UUID) (uuid.UUID, error) {
	var studentID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `SELECT id FROM students WHERE tenant_id=$1 AND user_id=$2 AND deleted_at IS NULL LIMIT 1`, tenantID, userID).Scan(&studentID)
	if err != nil {
		return uuid.Nil, errors.New("student profile not found")
	}
	return studentID, nil
}

func stringIndex(value int) string {
	return strconv.Itoa(value)
}
