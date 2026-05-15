package grading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	ManualScore(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, ManualScoreRequest) (SessionGradesResponse, error)
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

func (s *service) ManualScore(ctx context.Context, tenantID, gradingID, reviewerUserID uuid.UUID, req ManualScoreRequest) (SessionGradesResponse, error) {
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
