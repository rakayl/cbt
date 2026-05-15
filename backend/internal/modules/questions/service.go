package questions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
)

type Service interface {
	List(context.Context, uuid.UUID, pagination.Query, uuid.UUID, []string) (shared.ListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) (QuestionView, error)
	Usage(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) (QuestionUsageView, error)
	Versions(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) (QuestionVersionListResult, error)
	Create(context.Context, uuid.UUID, uuid.UUID, []string, CreateQuestionRequest) (QuestionView, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string, UpdateQuestionRequest) (QuestionView, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) error
	UploadMedia(context.Context, uuid.UUID, uuid.UUID, []string, uuid.UUID, *uuid.UUID, string, string, string, io.Reader, int64) (QuestionMediaView, error)
	DeleteMedia(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) error
	MediaContent(context.Context, uuid.UUID, uuid.UUID) (io.ReadCloser, string, error)
}

type service struct {
	repo Repository
	deps shared.Deps
}

func NewService(repo Repository, deps shared.Deps) Service { return &service{repo: repo, deps: deps} }

func (s *service) List(ctx context.Context, tenantID uuid.UUID, q pagination.Query, actorUserID uuid.UUID, permissions []string) (shared.ListResult, error) {
	out, err := s.listRecords(ctx, tenantID, q, actorUserID, permissions)
	if err != nil {
		return out, err
	}
	for index := range out.Items {
		tags, err := s.tags(ctx, out.Items[index].ID)
		if err != nil {
			return out, err
		}
		if out.Items[index].Metadata == nil {
			out.Items[index].Metadata = map[string]any{}
		}
		out.Items[index].Metadata["tags"] = tags
		out.Items[index].Metadata["tag_ids"] = tagIDs(tags)
	}
	return out, nil
}

func (s *service) listRecords(ctx context.Context, tenantID uuid.UUID, q pagination.Query, actorUserID uuid.UUID, permissions []string) (shared.ListResult, error) {
	if canManageAllQuestionOwners(permissions) {
		return s.repo.List(ctx, tenantID, q)
	}
	search := "%" + strings.ToLower(q.Search) + "%"
	where := "deleted_at IS NULL AND tenant_id=$1 AND owner_user_id=$2"
	args := []any{tenantID, actorUserID}
	nextArg := 3
	if q.Search != "" {
		where += fmt.Sprintf(" AND (lower(name) LIKE $%d OR lower(code) LIKE $%d)", nextArg, nextArg)
		args = append(args, search)
		nextArg++
	}
	var total int64
	if err := s.deps.DB.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM questions WHERE %s", where), args...).Scan(&total); err != nil {
		return shared.ListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	rows, err := s.deps.DB.Query(ctx, fmt.Sprintf(`
		SELECT id, tenant_id, code, name, COALESCE(description,''), status, metadata, created_at, updated_at, deleted_at
		FROM questions
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, nextArg, nextArg+1), args...)
	if err != nil {
		return shared.ListResult{}, err
	}
	defer rows.Close()
	items := []shared.Record{}
	for rows.Next() {
		var rec shared.Record
		var raw []byte
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.Code, &rec.Name, &rec.Description, &rec.Status, &raw, &rec.CreatedAt, &rec.UpdatedAt, &rec.DeletedAt); err != nil {
			return shared.ListResult{}, err
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &rec.Metadata)
		}
		items = append(items, rec)
	}
	return shared.ListResult{Items: items, Page: q.Page, Limit: q.Limit, Total: total}, rows.Err()
}

func (s *service) Get(ctx context.Context, tenantID, id, actorUserID uuid.UUID, permissions []string) (QuestionView, error) {
	if err := s.ensureQuestionReadable(ctx, tenantID, id, actorUserID, permissions); err != nil {
		return QuestionView{}, err
	}
	return s.get(ctx, tenantID, id)
}

func (s *service) get(ctx context.Context, tenantID, id uuid.UUID) (QuestionView, error) {
	var out QuestionView
	var raw []byte
	var lecturerID, ownerUserID string
	err := s.deps.DB.QueryRow(ctx, `
		SELECT id, tenant_id, code, name, question_bank_id, coalesce(lecturer_id::text,''), coalesce(owner_user_id::text,''),
		       content, question_type, answer_mode, difficulty, score, coalesce(explanation,''), status, metadata
		FROM questions
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		id, tenantID).Scan(&out.ID, &out.TenantID, &out.Code, &out.Name, &out.QuestionBankID, &lecturerID, &ownerUserID, &out.QuestionText, &out.QuestionType, &out.AnswerMode, &out.Difficulty, &out.Score, &out.Explanation, &out.Status, &raw)
	if err != nil {
		return QuestionView{}, err
	}
	if parsed, err := uuid.Parse(lecturerID); err == nil && parsed != uuid.Nil {
		out.LecturerID = &parsed
	}
	if parsed, err := uuid.Parse(ownerUserID); err == nil && parsed != uuid.Nil {
		out.OwnerUserID = &parsed
	}
	_ = json.Unmarshal(raw, &out.Metadata)
	options, err := s.options(ctx, id)
	if err != nil {
		return QuestionView{}, err
	}
	out.Options = options
	media, err := s.media(ctx, tenantID, id, nil, "question")
	if err != nil {
		return QuestionView{}, err
	}
	out.Media = media
	tags, err := s.tags(ctx, id)
	if err != nil {
		return QuestionView{}, err
	}
	out.Tags = tags
	out.TagIDs = tagIDs(tags)
	return out, nil
}

func (s *service) Usage(ctx context.Context, tenantID, id, actorUserID uuid.UUID, permissions []string) (QuestionUsageView, error) {
	if err := s.ensureQuestionReadable(ctx, tenantID, id, actorUserID, permissions); err != nil {
		return QuestionUsageView{}, err
	}
	out := QuestionUsageView{
		QuestionID:        id,
		CanHardDelete:     false,
		RecommendedAction: "archive",
	}
	_ = s.deps.DB.QueryRow(ctx, `
		SELECT COALESCE(version,1)
		FROM questions
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID).Scan(&out.Version)
	if err := s.deps.DB.QueryRow(ctx, `
		SELECT count(DISTINCT esq.exam_session_id)
		FROM exam_session_questions esq
		WHERE esq.tenant_id=$1 AND esq.question_id=$2 AND esq.deleted_at IS NULL`,
		tenantID, id).Scan(&out.ExamSessionCount); err != nil {
		return out, err
	}
	if err := s.deps.DB.QueryRow(ctx, `
		SELECT count(DISTINCT es.exam_id)
		FROM exam_session_questions esq
		JOIN exam_sessions es ON es.id=esq.exam_session_id AND es.deleted_at IS NULL
		WHERE esq.tenant_id=$1 AND esq.question_id=$2 AND esq.deleted_at IS NULL`,
		tenantID, id).Scan(&out.ExamCount); err != nil {
		return out, err
	}
	if err := s.deps.DB.QueryRow(ctx, `
		SELECT count(*)
		FROM answers
		WHERE tenant_id=$1 AND question_id=$2 AND deleted_at IS NULL`,
		tenantID, id).Scan(&out.AnswerCount); err != nil {
		return out, err
	}
	if err := s.deps.DB.QueryRow(ctx, `
		SELECT count(*)
		FROM grading
		WHERE tenant_id=$1 AND metadata->>'question_id'=$2 AND deleted_at IS NULL`,
		tenantID, id.String()).Scan(&out.GradingCount); err != nil {
		return out, err
	}
	out.CanHardDelete = out.ExamSessionCount == 0 && out.AnswerCount == 0 && out.GradingCount == 0
	if out.CanHardDelete {
		out.RecommendedAction = "soft_delete"
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT esq.exam_session_id
		FROM exam_session_questions esq
		WHERE esq.tenant_id=$1 AND esq.question_id=$2 AND esq.deleted_at IS NULL
		ORDER BY esq.created_at DESC
		LIMIT 20`, tenantID, id)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var sessionID uuid.UUID
		if err := rows.Scan(&sessionID); err != nil {
			rows.Close()
			return out, err
		}
		out.UsedInExamSessionIDs = append(out.UsedInExamSessionIDs, sessionID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	rows, err = s.deps.DB.Query(ctx, `
		SELECT e.id, e.name, count(DISTINCT es.id), max(es.started_at)
		FROM exam_session_questions esq
		JOIN exam_sessions es ON es.id=esq.exam_session_id AND es.deleted_at IS NULL
		JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
		WHERE esq.tenant_id=$1 AND esq.question_id=$2 AND esq.deleted_at IS NULL
		GROUP BY e.id, e.name
		ORDER BY max(es.started_at) DESC
		LIMIT 20`, tenantID, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item QuestionExamRef
		var lastStarted *time.Time
		if err := rows.Scan(&item.ExamID, &item.ExamName, &item.SessionCount, &lastStarted); err != nil {
			return out, err
		}
		if lastStarted != nil {
			item.LastStartedAt = lastStarted.UTC().Format(time.RFC3339)
		}
		out.UsedInExamSummaries = append(out.UsedInExamSummaries, item)
	}
	return out, rows.Err()
}

func (s *service) Versions(ctx context.Context, tenantID, questionID, actorUserID uuid.UUID, permissions []string) (QuestionVersionListResult, error) {
	if err := s.ensureQuestionReadable(ctx, tenantID, questionID, actorUserID, permissions); err != nil {
		return QuestionVersionListResult{}, err
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT id, question_id, version, actor_user_id, event_type, snapshot, metadata, created_at
		FROM question_versions
		WHERE tenant_id=$1 AND question_id=$2 AND deleted_at IS NULL
		ORDER BY version DESC, created_at DESC`,
		tenantID, questionID)
	if err != nil {
		return QuestionVersionListResult{}, err
	}
	defer rows.Close()
	out := QuestionVersionListResult{Items: []QuestionVersionView{}}
	for rows.Next() {
		var item QuestionVersionView
		var actorUserID *uuid.UUID
		var rawSnapshot, rawMetadata []byte
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.QuestionID, &item.Version, &actorUserID, &item.EventType, &rawSnapshot, &rawMetadata, &createdAt); err != nil {
			return QuestionVersionListResult{}, err
		}
		item.ActorUserID = actorUserID
		item.Snapshot = map[string]any{}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(rawSnapshot, &item.Snapshot)
		_ = json.Unmarshal(rawMetadata, &item.Metadata)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out.Items = append(out.Items, item)
	}
	if err := rows.Err(); err != nil {
		return QuestionVersionListResult{}, err
	}
	out.Total = len(out.Items)
	return out, nil
}

func (s *service) Create(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, req CreateQuestionRequest) (QuestionView, error) {
	if err := ValidateCreate(req); err != nil {
		return QuestionView{}, err
	}
	if err := validateQuestion(req.AnswerMode, req.Options); err != nil {
		return QuestionView{}, err
	}
	lecturerID, ownerUserID, err := s.resolveQuestionOwner(ctx, tenantID, actorUserID, permissions, req.LecturerID)
	if err != nil {
		return QuestionView{}, err
	}
	if err := s.ensureQuestionBankAssignable(ctx, tenantID, req.QuestionBankID, ownerUserID); err != nil {
		return QuestionView{}, err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return QuestionView{}, err
	}
	defer tx.Rollback(ctx)
	id := uuid.New()
	status := req.Status
	if status == "" {
		status = "draft"
	}
	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}
	score := req.Score
	if score <= 0 {
		score = 1
	}
	meta := req.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	meta["answer_mode"] = req.AnswerMode
	meta["question_type"] = "multiple_choice"
	meta["lecturer_id"] = lecturerID.String()
	meta["owner_user_id"] = ownerUserID.String()
	meta["tag_ids"] = req.TagIDs
	meta["version"] = 1
	raw, _ := json.Marshal(meta)
	_, err = tx.Exec(ctx, `
		INSERT INTO questions(id,tenant_id,code,name,description,status,metadata,question_bank_id,lecturer_id,owner_user_id,question_type,answer_mode,difficulty,content,explanation,score)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'multiple_choice',$11,$12,$13,$14,$15)`,
		id, tenantID, req.Code, req.Name, shortText(req.QuestionText), status, raw, req.QuestionBankID, lecturerID, ownerUserID, req.AnswerMode, difficulty, req.QuestionText, req.Explanation, score)
	if err != nil {
		return QuestionView{}, err
	}
	if err = insertOptions(ctx, tx, tenantID, id, req.Options); err != nil {
		return QuestionView{}, err
	}
	if err = syncQuestionTags(ctx, tx, tenantID, id, req.TagIDs, ownerUserID, canManageAllQuestionTags(permissions)); err != nil {
		return QuestionView{}, err
	}
	if err = s.saveQuestionVersion(ctx, tx, tenantID, id, 1, actorUserID, "create"); err != nil {
		return QuestionView{}, err
	}
	if err = insertAcademicAudit(ctx, tx, tenantID, actorUserID, "question.create", id, map[string]any{
		"version": 1,
		"code":    req.Code,
		"name":    req.Name,
	}); err != nil {
		return QuestionView{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return QuestionView{}, err
	}
	out, err := s.get(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "questions.created", body)
	}
	return out, err
}

func (s *service) Update(ctx context.Context, tenantID, id, actorUserID uuid.UUID, permissions []string, req UpdateQuestionRequest) (QuestionView, error) {
	if err := s.ensureQuestionWritable(ctx, tenantID, id, actorUserID, permissions); err != nil {
		return QuestionView{}, err
	}
	if err := ValidateUpdate(req); err != nil {
		return QuestionView{}, err
	}
	if err := validateQuestion(req.AnswerMode, req.Options); err != nil {
		return QuestionView{}, err
	}
	lecturerID, ownerUserID, err := s.resolveQuestionOwner(ctx, tenantID, actorUserID, permissions, req.LecturerID)
	if err != nil {
		return QuestionView{}, err
	}
	if err := s.ensureQuestionBankAssignable(ctx, tenantID, req.QuestionBankID, ownerUserID); err != nil {
		return QuestionView{}, err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return QuestionView{}, err
	}
	defer tx.Rollback(ctx)
	meta := req.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	meta["answer_mode"] = req.AnswerMode
	meta["question_type"] = "multiple_choice"
	meta["lecturer_id"] = lecturerID.String()
	meta["owner_user_id"] = ownerUserID.String()
	meta["tag_ids"] = req.TagIDs
	var previousVersion int
	var previousRaw []byte
	_ = tx.QueryRow(ctx, `SELECT COALESCE(version,1), COALESCE(metadata,'{}'::jsonb) FROM questions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, id, tenantID).Scan(&previousVersion, &previousRaw)
	nextVersion := previousVersion + 1
	meta["version"] = nextVersion
	raw, _ := json.Marshal(meta)
	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}
	score := req.Score
	if score <= 0 {
		score = 1
	}
	_, err = tx.Exec(ctx, `
		UPDATE questions
		SET code=$1, name=$2, description=$3, status=$4, metadata=$5, question_bank_id=$6,
		    lecturer_id=$7, owner_user_id=$8, question_type='multiple_choice', answer_mode=$9,
		    difficulty=$10, content=$11, explanation=$12, score=$13, version=$16, updated_at=now()
		WHERE id=$14 AND tenant_id=$15 AND deleted_at IS NULL`,
		req.Code, req.Name, shortText(req.QuestionText), req.Status, raw, req.QuestionBankID, lecturerID, ownerUserID, req.AnswerMode, difficulty, req.QuestionText, req.Explanation, score, id, tenantID, nextVersion)
	if err != nil {
		return QuestionView{}, err
	}
	if err = syncOptions(ctx, tx, tenantID, id, req.Options); err != nil {
		return QuestionView{}, err
	}
	if err = syncQuestionTags(ctx, tx, tenantID, id, req.TagIDs, ownerUserID, canManageAllQuestionTags(permissions)); err != nil {
		return QuestionView{}, err
	}
	if err = s.saveQuestionVersion(ctx, tx, tenantID, id, nextVersion, actorUserID, "update"); err != nil {
		return QuestionView{}, err
	}
	if err = insertAcademicAudit(ctx, tx, tenantID, actorUserID, "question.update", id, map[string]any{
		"previous_version":  previousVersion,
		"new_version":       nextVersion,
		"previous_metadata": json.RawMessage(previousRaw),
		"code":              req.Code,
		"name":              req.Name,
	}); err != nil {
		return QuestionView{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return QuestionView{}, err
	}
	out, err := s.get(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "questions.updated", body)
	}
	return out, err
}

func (s *service) Delete(ctx context.Context, tenantID, actorUserID, id uuid.UUID, permissions []string) error {
	if err := s.ensureQuestionWritable(ctx, tenantID, id, actorUserID, permissions); err != nil {
		return err
	}
	usage, err := s.Usage(ctx, tenantID, id, actorUserID, permissions)
	if err != nil {
		return err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	if usage.CanHardDelete {
		status = "retired"
	} else {
		status = "archived"
	}
	archiveMeta, _ := json.Marshal(map[string]any{
		"delete_policy":       "soft_delete_only",
		"archived_at":         time.Now().UTC().Format(time.RFC3339Nano),
		"archived_by_user_id": actorUserID.String(),
		"usage":               usage,
	})
	_, err = tx.Exec(ctx, `
		UPDATE questions
		SET status=$1,
		    metadata=COALESCE(metadata,'{}'::jsonb) || $2::jsonb,
		    deleted_at=now(),
		    updated_at=now()
		WHERE id=$3 AND tenant_id=$4 AND deleted_at IS NULL`,
		status, archiveMeta, id, tenantID)
	if err != nil {
		return err
	}
	_, _ = tx.Exec(ctx, `UPDATE question_options SET deleted_at=now(), updated_at=now() WHERE question_id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, id, tenantID)
	_, _ = tx.Exec(ctx, `UPDATE question_tag_relations SET deleted_at=now(), updated_at=now() WHERE question_id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, id, tenantID)
	_, _ = tx.Exec(ctx, `UPDATE question_media SET deleted_at=now(), updated_at=now() WHERE question_id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, id, tenantID)
	if err = insertAcademicAudit(ctx, tx, tenantID, actorUserID, "question.archive", id, map[string]any{
		"status": status,
		"usage":  usage,
	}); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "questions.deleted", []byte(id.String()))
	}
	return nil
}

func (s *service) saveQuestionVersion(ctx context.Context, tx pgx.Tx, tenantID, questionID uuid.UUID, version int, actorUserID uuid.UUID, eventType string) error {
	var snapshot []byte
	err := tx.QueryRow(ctx, `
		SELECT jsonb_build_object(
			'id', q.id,
			'tenant_id', q.tenant_id,
			'code', q.code,
			'name', q.name,
			'description', COALESCE(q.description,''),
			'question_bank_id', q.question_bank_id,
			'lecturer_id', q.lecturer_id,
			'owner_user_id', q.owner_user_id,
			'question_type', COALESCE(q.question_type, 'multiple_choice'),
			'answer_mode', COALESCE(q.answer_mode, 'single'),
			'difficulty', COALESCE(q.difficulty, 'medium'),
			'content', COALESCE(q.content, ''),
			'explanation', COALESCE(q.explanation, ''),
			'score', CASE WHEN COALESCE(q.score,0) > 0 THEN q.score ELSE 1 END,
			'status', q.status,
			'version', COALESCE(q.version, $3),
			'metadata', COALESCE(q.metadata,'{}'::jsonb),
			'options', COALESCE((
				SELECT jsonb_agg(
					jsonb_build_object(
						'id', qo.id,
						'label', qo.code,
						'text', COALESCE(qo.description,''),
						'is_correct', qo.is_correct,
						'sort_order', qo.sort_order,
						'metadata', COALESCE(qo.metadata,'{}'::jsonb)
					)
					ORDER BY qo.sort_order ASC, qo.created_at ASC
				)
				FROM question_options qo
				WHERE qo.question_id=q.id AND qo.tenant_id=q.tenant_id AND qo.deleted_at IS NULL
			), '[]'::jsonb),
			'tags', COALESCE((
				SELECT jsonb_agg(
					jsonb_build_object(
						'id', qt.id,
						'name', qt.name,
						'owner_user_id', qt.owner_user_id,
						'lecturer_id', qt.lecturer_id
					)
					ORDER BY qt.name ASC
				)
				FROM question_tag_relations qtr
				JOIN question_tags qt ON qt.id=qtr.question_tag_id AND qt.deleted_at IS NULL
				WHERE qtr.question_id=q.id AND qtr.deleted_at IS NULL
			), '[]'::jsonb),
			'media', COALESCE((
				SELECT jsonb_agg(
					jsonb_build_object(
						'id', qm.id,
						'option_id', qm.option_id,
						'usage_type', qm.usage_type,
						'object_key', qm.object_key,
						'mime_type', qm.mime_type,
						'file_size', qm.file_size,
						'sort_order', qm.sort_order
					)
					ORDER BY qm.usage_type ASC, qm.sort_order ASC, qm.created_at ASC
				)
				FROM question_media qm
				WHERE qm.question_id=q.id AND qm.tenant_id=q.tenant_id AND qm.deleted_at IS NULL
			), '[]'::jsonb),
			'captured_at', now()
		)
		FROM questions q
		WHERE q.id=$1 AND q.tenant_id=$2`,
		questionID, tenantID, version).Scan(&snapshot)
	if err != nil {
		return err
	}
	metadata, _ := json.Marshal(map[string]any{
		"event_type":    eventType,
		"actor_user_id": actorUserID.String(),
		"captured_at":   time.Now().UTC().Format(time.RFC3339Nano),
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO question_versions(id,tenant_id,question_id,version,actor_user_id,event_type,snapshot,metadata)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (question_id, version) WHERE deleted_at IS NULL
		DO UPDATE SET snapshot=excluded.snapshot, metadata=excluded.metadata, updated_at=now()`,
		uuid.New(), tenantID, questionID, version, actorUserID, eventType, snapshot, metadata)
	return err
}

func bumpQuestionVersion(ctx context.Context, tx pgx.Tx, tenantID, questionID uuid.UUID) (int, error) {
	var version int
	err := tx.QueryRow(ctx, `
		UPDATE questions
		SET version=COALESCE(version,1)+1,
		    metadata=COALESCE(metadata,'{}'::jsonb) || jsonb_build_object('version', COALESCE(version,1)+1),
		    updated_at=now()
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL
		RETURNING version`,
		questionID, tenantID).Scan(&version)
	return version, err
}

func insertAcademicAudit(ctx context.Context, tx pgx.Tx, tenantID, actorUserID uuid.UUID, eventType string, entityID uuid.UUID, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["actor_user_id"] = actorUserID.String()
	metadata["entity_id"] = entityID.String()
	metadata["entity_type"] = "question"
	metadata["event_type"] = eventType
	metadata["recorded_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	raw, _ := json.Marshal(metadata)
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs(id,tenant_id,code,name,description,status,metadata)
		VALUES($1,$2,$3,$4,$5,'active',$6)`,
		uuid.New(), tenantID, "AUD-"+entityID.String()[:8], eventType, "Academic audit trail", raw)
	return err
}

func syncQuestionTags(ctx context.Context, tx pgx.Tx, tenantID, questionID uuid.UUID, tagIDs []uuid.UUID, ownerUserID uuid.UUID, allowAny bool) error {
	_, err := tx.Exec(ctx, `UPDATE question_tag_relations SET deleted_at=now(), updated_at=now() WHERE question_id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, questionID, tenantID)
	if err != nil {
		return err
	}
	seen := map[uuid.UUID]bool{}
	for _, tagID := range tagIDs {
		if tagID == uuid.Nil || seen[tagID] {
			continue
		}
		seen[tagID] = true
		var tagName string
		if err := tx.QueryRow(ctx, `
			SELECT name
			FROM question_tags
			WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL
			  AND ($3 OR owner_user_id=$4)`,
			tagID, tenantID, allowAny, ownerUserID).Scan(&tagName); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO question_tag_relations(id,tenant_id,code,name,status,question_id,question_tag_id,metadata)
			VALUES($1,$2,$3,$4,'active',$5,$6,'{}')`,
			uuid.New(), tenantID, "QTR-"+questionID.String()[:8]+"-"+tagID.String()[:8], tagName, questionID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertOptions(ctx context.Context, tx pgx.Tx, tenantID, questionID uuid.UUID, options []QuestionOptionInput) error {
	for index, option := range options {
		meta, _ := json.Marshal(map[string]any{"label": option.Label, "text": option.Text})
		optionID := uuid.New()
		if option.ID != nil && *option.ID != uuid.Nil {
			optionID = *option.ID
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO question_options(id,tenant_id,code,name,description,status,metadata,question_id,is_correct,sort_order)
			VALUES($1,$2,$3,$4,$5,'active',$6,$7,$8,$9)`,
			optionID, tenantID, strings.ToUpper(option.Label), strings.ToUpper(option.Label), option.Text, meta, questionID, option.IsCorrect, index+1)
		if err != nil {
			return err
		}
	}
	return nil
}

func syncOptions(ctx context.Context, tx pgx.Tx, tenantID, questionID uuid.UUID, options []QuestionOptionInput) error {
	keep := []uuid.UUID{}
	for index, option := range options {
		label := strings.ToUpper(strings.TrimSpace(option.Label))
		meta, _ := json.Marshal(map[string]any{"label": label, "text": option.Text})
		if option.ID != nil && *option.ID != uuid.Nil {
			var exists bool
			if err := tx.QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM question_options
					WHERE id=$1 AND tenant_id=$2 AND question_id=$3 AND deleted_at IS NULL
				)`, *option.ID, tenantID, questionID).Scan(&exists); err != nil {
				return err
			}
			if exists {
				_, err := tx.Exec(ctx, `
					UPDATE question_options
					SET code=$1, name=$1, description=$2, metadata=$3, is_correct=$4, sort_order=$5, updated_at=now()
					WHERE id=$6 AND tenant_id=$7 AND question_id=$8 AND deleted_at IS NULL`,
					label, option.Text, meta, option.IsCorrect, index+1, *option.ID, tenantID, questionID)
				if err != nil {
					return err
				}
				keep = append(keep, *option.ID)
				continue
			}
		}
		optionID := uuid.New()
		_, err := tx.Exec(ctx, `
			INSERT INTO question_options(id,tenant_id,code,name,description,status,metadata,question_id,is_correct,sort_order)
			VALUES($1,$2,$3,$3,$4,'active',$5,$6,$7,$8)`,
			optionID, tenantID, label, option.Text, meta, questionID, option.IsCorrect, index+1)
		if err != nil {
			return err
		}
		keep = append(keep, optionID)
	}
	rows, err := tx.Query(ctx, `
		SELECT id
		FROM question_options
		WHERE tenant_id=$1 AND question_id=$2 AND deleted_at IS NULL AND NOT (id = ANY($3::uuid[]))`,
		tenantID, questionID, keep)
	if err != nil {
		return err
	}
	removed := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		removed = append(removed, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(removed) > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE question_media
			SET deleted_at=now(), updated_at=now()
			WHERE tenant_id=$1 AND option_id = ANY($2::uuid[]) AND deleted_at IS NULL`,
			tenantID, removed); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE question_options
		SET deleted_at=now(), updated_at=now()
		WHERE tenant_id=$1 AND question_id=$2 AND deleted_at IS NULL AND NOT (id = ANY($3::uuid[]))`,
		tenantID, questionID, keep)
	return err
}

func (s *service) options(ctx context.Context, questionID uuid.UUID) ([]QuestionOptionView, error) {
	rows, err := s.deps.DB.Query(ctx, `
		SELECT id, code, description, is_correct, sort_order, metadata
		FROM question_options
		WHERE question_id=$1 AND deleted_at IS NULL
		ORDER BY sort_order ASC`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QuestionOptionView{}
	for rows.Next() {
		var item QuestionOptionView
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Label, &item.Text, &item.IsCorrect, &item.SortOrder, &raw); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &item.Metadata)
		media, err := s.media(ctx, uuid.Nil, questionID, &item.ID, "option")
		if err != nil {
			return nil, err
		}
		item.Media = media
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) UploadMedia(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, questionID uuid.UUID, optionID *uuid.UUID, usageType, filename, contentType string, reader io.Reader, size int64) (QuestionMediaView, error) {
	if s.deps.Storage == nil {
		return QuestionMediaView{}, errors.New("object storage is not configured")
	}
	if err := s.ensureQuestionWritable(ctx, tenantID, questionID, actorUserID, permissions); err != nil {
		return QuestionMediaView{}, err
	}
	usageType = strings.TrimSpace(usageType)
	if usageType == "" {
		usageType = "question"
	}
	if usageType != "question" && usageType != "option" && usageType != "explanation" {
		return QuestionMediaView{}, errors.New("unsupported media usage type")
	}
	if usageType == "option" {
		if optionID == nil || *optionID == uuid.Nil {
			return QuestionMediaView{}, errors.New("option_id is required for option media")
		}
		var exists bool
		err := s.deps.DB.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM question_options
				WHERE id=$1 AND tenant_id=$2 AND question_id=$3 AND deleted_at IS NULL
			)`, *optionID, tenantID, questionID).Scan(&exists)
		if err != nil {
			return QuestionMediaView{}, err
		}
		if !exists {
			return QuestionMediaView{}, errors.New("option does not belong to question")
		}
	}
	if size > 5*1024*1024 {
		return QuestionMediaView{}, errors.New("image size exceeds 5MB")
	}
	payload, err := io.ReadAll(io.LimitReader(reader, 5*1024*1024+1))
	if err != nil {
		return QuestionMediaView{}, err
	}
	if len(payload) == 0 {
		return QuestionMediaView{}, errors.New("empty image")
	}
	if len(payload) > 5*1024*1024 {
		return QuestionMediaView{}, errors.New("image size exceeds 5MB")
	}
	detected := http.DetectContentType(payload)
	if detected != "image/jpeg" && detected != "image/png" && detected != "image/webp" {
		return QuestionMediaView{}, errors.New("only jpeg, png, and webp images are allowed")
	}
	extension := ".jpg"
	if detected == "image/png" {
		extension = ".png"
	}
	if detected == "image/webp" {
		extension = ".webp"
	}
	mediaID := uuid.New()
	folder := "question"
	if usageType == "option" && optionID != nil {
		folder = "options/" + optionID.String()
	}
	if usageType == "explanation" {
		folder = "explanation"
	}
	objectKey := filepath.ToSlash(fmt.Sprintf("tenants/%s/questions/%s/%s/%s%s", tenantID, questionID, folder, mediaID, extension))
	_, err = s.deps.Storage.PutObject(ctx, s.deps.Config.S3Bucket, objectKey, bytes.NewReader(payload), int64(len(payload)), minio.PutObjectOptions{ContentType: detected})
	if err != nil {
		return QuestionMediaView{}, err
	}
	var optionArg any
	if optionID != nil {
		optionArg = *optionID
	}
	var sortOrder int
	_ = s.deps.DB.QueryRow(ctx, `
		SELECT COALESCE(max(sort_order), 0) + 1
		FROM question_media
		WHERE tenant_id=$1 AND question_id=$2 AND usage_type=$3 AND (($4::uuid IS NULL AND option_id IS NULL) OR option_id=$4) AND deleted_at IS NULL`,
		tenantID, questionID, usageType, optionArg).Scan(&sortOrder)
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return QuestionMediaView{}, err
	}
	defer tx.Rollback(ctx)
	meta, _ := json.Marshal(map[string]any{"uploaded_at": time.Now().UTC(), "source_content_type": contentType})
	_, err = tx.Exec(ctx, `
		INSERT INTO question_media(id,tenant_id,question_id,option_id,media_type,usage_type,object_key,original_filename,mime_type,file_size,sort_order,metadata)
		VALUES($1,$2,$3,$4,'image',$5,$6,$7,$8,$9,$10,$11)`,
		mediaID, tenantID, questionID, optionID, usageType, objectKey, filepath.Base(filename), detected, len(payload), sortOrder, meta)
	if err != nil {
		return QuestionMediaView{}, err
	}
	nextVersion, err := bumpQuestionVersion(ctx, tx, tenantID, questionID)
	if err != nil {
		return QuestionMediaView{}, err
	}
	if err = s.saveQuestionVersion(ctx, tx, tenantID, questionID, nextVersion, actorUserID, "media_upload"); err != nil {
		return QuestionMediaView{}, err
	}
	if err = insertAcademicAudit(ctx, tx, tenantID, actorUserID, "question.media_upload", questionID, map[string]any{
		"version":    nextVersion,
		"media_id":   mediaID.String(),
		"usage_type": usageType,
		"option_id":  optionArg,
	}); err != nil {
		return QuestionMediaView{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return QuestionMediaView{}, err
	}
	items, err := s.media(ctx, tenantID, questionID, optionID, usageType)
	if err != nil {
		return QuestionMediaView{}, err
	}
	for _, item := range items {
		if item.ID == mediaID {
			return item, nil
		}
	}
	return QuestionMediaView{}, errors.New("media upload saved but cannot be loaded")
}

func (s *service) DeleteMedia(ctx context.Context, tenantID, actorUserID, mediaID uuid.UUID, permissions []string) error {
	var questionID uuid.UUID
	if err := s.deps.DB.QueryRow(ctx, `SELECT question_id FROM question_media WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, mediaID, tenantID).Scan(&questionID); err != nil {
		return err
	}
	if err := s.ensureQuestionWritable(ctx, tenantID, questionID, actorUserID, permissions); err != nil {
		return err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE question_media SET deleted_at=now(), updated_at=now() WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, mediaID, tenantID)
	if err != nil {
		return err
	}
	nextVersion, err := bumpQuestionVersion(ctx, tx, tenantID, questionID)
	if err != nil {
		return err
	}
	if err = s.saveQuestionVersion(ctx, tx, tenantID, questionID, nextVersion, actorUserID, "media_delete"); err != nil {
		return err
	}
	if err = insertAcademicAudit(ctx, tx, tenantID, actorUserID, "question.media_delete", questionID, map[string]any{
		"version":  nextVersion,
		"media_id": mediaID.String(),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *service) MediaContent(ctx context.Context, tenantID, mediaID uuid.UUID) (io.ReadCloser, string, error) {
	var objectKey, mimeType string
	if err := s.deps.DB.QueryRow(ctx, `SELECT object_key, mime_type FROM question_media WHERE id=$1 AND tenant_id=$2`, mediaID, tenantID).Scan(&objectKey, &mimeType); err != nil {
		return nil, "", err
	}
	if s.deps.Storage == nil {
		return nil, "", errors.New("object storage is not configured")
	}
	object, err := s.deps.Storage.GetObject(ctx, s.deps.Config.S3Bucket, objectKey, minio.GetObjectOptions{})
	return object, mimeType, err
}

func (s *service) media(ctx context.Context, tenantID, questionID uuid.UUID, optionID *uuid.UUID, usageType string) ([]QuestionMediaView, error) {
	args := []any{questionID, usageType}
	where := "question_id=$1 AND usage_type=$2 AND deleted_at IS NULL"
	if tenantID != uuid.Nil {
		where += " AND tenant_id=$3"
		args = append(args, tenantID)
	}
	if optionID == nil {
		where += " AND option_id IS NULL"
	} else {
		where += fmt.Sprintf(" AND option_id=$%d", len(args)+1)
		args = append(args, *optionID)
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT id, question_id, option_id, media_type, usage_type, object_key, COALESCE(original_filename,''), mime_type, file_size, width, height, sort_order
		FROM question_media
		WHERE `+where+`
		ORDER BY sort_order ASC, created_at ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QuestionMediaView{}
	for rows.Next() {
		var item QuestionMediaView
		if err := rows.Scan(&item.ID, &item.QuestionID, &item.OptionID, &item.MediaType, &item.UsageType, &item.ObjectKey, &item.OriginalFilename, &item.MimeType, &item.FileSize, &item.Width, &item.Height, &item.SortOrder); err != nil {
			return nil, err
		}
		item.URL = fmt.Sprintf("/questions/media/%s/content", item.ID)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) tags(ctx context.Context, questionID uuid.UUID) ([]QuestionTagView, error) {
	rows, err := s.deps.DB.Query(ctx, `
		SELECT qt.id, qt.code, qt.name, COALESCE(qt.color,''),
		       COALESCE(qt.lecturer_id::text,''), COALESCE(qt.owner_user_id::text,''), COALESCE(l.name,'')
		FROM question_tag_relations qtr
		JOIN question_tags qt ON qt.id=qtr.question_tag_id AND qt.deleted_at IS NULL
		LEFT JOIN lecturers l ON l.id=qt.lecturer_id AND l.deleted_at IS NULL
		WHERE qtr.question_id=$1 AND qtr.deleted_at IS NULL
		ORDER BY qt.name ASC`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QuestionTagView{}
	for rows.Next() {
		var item QuestionTagView
		var lecturerID, ownerUserID string
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Color, &lecturerID, &ownerUserID, &item.LecturerName); err != nil {
			return nil, err
		}
		if parsed, err := uuid.Parse(lecturerID); err == nil && parsed != uuid.Nil {
			item.LecturerID = &parsed
		}
		if parsed, err := uuid.Parse(ownerUserID); err == nil && parsed != uuid.Nil {
			item.OwnerUserID = &parsed
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func tagIDs(tags []QuestionTagView) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(tags))
	for _, tag := range tags {
		out = append(out, tag.ID)
	}
	return out
}

func validateQuestion(answerMode string, options []QuestionOptionInput) error {
	if len(options) < 2 {
		return errors.New("minimal 2 opsi jawaban")
	}
	correct := 0
	labels := map[string]bool{}
	for _, option := range options {
		label := strings.ToUpper(strings.TrimSpace(option.Label))
		if labels[label] {
			return fmt.Errorf("label opsi duplikat: %s", label)
		}
		labels[label] = true
		if option.IsCorrect {
			correct++
		}
	}
	if answerMode == "single" && correct != 1 {
		return errors.New("single answer wajib punya tepat 1 jawaban benar")
	}
	if answerMode == "multiple" && correct < 1 {
		return errors.New("multiple answer wajib punya minimal 1 jawaban benar")
	}
	return nil
}

func (s *service) ensureQuestionWritable(ctx context.Context, tenantID, questionID, actorUserID uuid.UUID, permissions []string) error {
	if canManageAllQuestionOwners(permissions) {
		var exists bool
		err := s.deps.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM questions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)`, questionID, tenantID).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("question not found")
		}
		return nil
	}
	var ownerUserID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `SELECT owner_user_id FROM questions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, questionID, tenantID).Scan(&ownerUserID)
	if err != nil {
		return err
	}
	if ownerUserID != actorUserID {
		return errors.New("question belongs to another lecturer")
	}
	return nil
}

func (s *service) ensureQuestionReadable(ctx context.Context, tenantID, questionID, actorUserID uuid.UUID, permissions []string) error {
	if canManageAllQuestionOwners(permissions) {
		var exists bool
		err := s.deps.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM questions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)`, questionID, tenantID).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("question not found")
		}
		return nil
	}
	var ownerUserID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `
		SELECT COALESCE(owner_user_id,'00000000-0000-0000-0000-000000000000')
		FROM questions
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, questionID, tenantID).Scan(&ownerUserID)
	if err != nil {
		return err
	}
	if ownerUserID != actorUserID {
		return errors.New("question belongs to another lecturer")
	}
	return nil
}

func (s *service) ensureQuestionBankAssignable(ctx context.Context, tenantID, questionBankID, ownerUserID uuid.UUID) error {
	var bankOwnerUserID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `
		SELECT COALESCE(owner_user_id,'00000000-0000-0000-0000-000000000000')
		FROM question_banks
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, questionBankID, tenantID).Scan(&bankOwnerUserID)
	if err != nil {
		return errors.New("bank soal tidak ditemukan")
	}
	if bankOwnerUserID != uuid.Nil && bankOwnerUserID != ownerUserID {
		return errors.New("bank soal dimiliki guru lain")
	}
	return nil
}

func (s *service) resolveQuestionOwner(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, requestedLecturerID *uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	if canManageAllQuestionOwners(permissions) {
		if requestedLecturerID == nil || *requestedLecturerID == uuid.Nil {
			return uuid.Nil, uuid.Nil, errors.New("admin wajib memilih guru pemilik soal")
		}
		var ownerUserID uuid.UUID
		err := s.deps.DB.QueryRow(ctx, `
			SELECT user_id
			FROM lecturers
			WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL AND status='active'
			LIMIT 1`,
			*requestedLecturerID, tenantID).Scan(&ownerUserID)
		if err != nil {
			return uuid.Nil, uuid.Nil, errors.New("guru tidak ditemukan atau belum punya akun")
		}
		if ownerUserID == uuid.Nil {
			return uuid.Nil, uuid.Nil, errors.New("guru belum terhubung dengan akun user")
		}
		return *requestedLecturerID, ownerUserID, nil
	}

	var lecturerID, ownerUserID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `
		SELECT id, user_id
		FROM lecturers
		WHERE user_id=$1 AND tenant_id=$2 AND deleted_at IS NULL AND status='active'
		LIMIT 1`,
		actorUserID, tenantID).Scan(&lecturerID, &ownerUserID)
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("akun login ini belum terhubung ke data guru")
	}
	if requestedLecturerID != nil && *requestedLecturerID != uuid.Nil && *requestedLecturerID != lecturerID {
		return uuid.Nil, uuid.Nil, errors.New("guru hanya boleh membuat soal atas nama dirinya sendiri")
	}
	return lecturerID, ownerUserID, nil
}

func hasPermission(permissions []string, permission string) bool {
	for _, item := range permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func canManageAllQuestionTags(permissions []string) bool {
	return canManageAllQuestionOwners(permissions)
}

func canManageAllQuestionOwners(permissions []string) bool {
	return hasPermission(permissions, "*") || hasPermission(permissions, "users:read") || hasPermission(permissions, "tenants:read")
}

func shortText(value string) string {
	clean := strings.TrimSpace(value)
	if len(clean) <= 160 {
		return clean
	}
	return clean[:160]
}
