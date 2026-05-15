package question_tags

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
)

type Service interface {
	List(context.Context, uuid.UUID, uuid.UUID, []string, pagination.Query) (shared.ListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID, []string, uuid.UUID) (shared.Record, error)
	Create(context.Context, uuid.UUID, uuid.UUID, []string, CreateQuestionTagRequest) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string, UpdateQuestionTagRequest) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) error
}

type service struct {
	repo Repository
	deps shared.Deps
}

func NewService(repo Repository, deps shared.Deps) Service { return &service{repo: repo, deps: deps} }

func (s *service) List(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, q pagination.Query) (shared.ListResult, error) {
	if q.Limit < 1 || q.Limit > 500 {
		q.Limit = 200
	}
	if q.Page < 1 {
		q.Page = 1
	}
	admin := canManageAllTags(permissions)
	search := "%" + strings.ToLower(strings.TrimSpace(q.Search)) + "%"
	args := []any{tenantID}
	where := "qt.tenant_id=$1 AND qt.deleted_at IS NULL"
	next := 2
	if !admin {
		where += fmt.Sprintf(" AND qt.owner_user_id=$%d", next)
		args = append(args, actorUserID)
		next++
	}
	if strings.TrimSpace(q.Search) != "" {
		where += fmt.Sprintf(" AND (lower(qt.name) LIKE $%d OR lower(qt.code) LIKE $%d OR lower(COALESCE(l.name,'')) LIKE $%d)", next, next, next)
		args = append(args, search)
		next++
	}
	var total int64
	if err := s.deps.DB.QueryRow(ctx, `
		SELECT count(*)
		FROM question_tags qt
		LEFT JOIN lecturers l ON l.id=qt.lecturer_id AND l.deleted_at IS NULL
		WHERE `+where, args...).Scan(&total); err != nil {
		return shared.ListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	rows, err := s.deps.DB.Query(ctx, `
		SELECT qt.id, qt.tenant_id, qt.code, qt.name, COALESCE(qt.description,''), qt.status, qt.metadata,
		       qt.created_at, qt.updated_at, qt.deleted_at,
		       COALESCE(qt.lecturer_id::text,''), COALESCE(qt.owner_user_id::text,''),
		       COALESCE(l.name,''), COALESCE(u.name,''), COALESCE(u.email,'')
		FROM question_tags qt
		LEFT JOIN lecturers l ON l.id=qt.lecturer_id AND l.deleted_at IS NULL
		LEFT JOIN users u ON u.id=qt.owner_user_id AND u.deleted_at IS NULL
		WHERE `+where+`
		ORDER BY COALESCE(l.name,''), qt.name
		LIMIT $`+fmt.Sprint(next)+` OFFSET $`+fmt.Sprint(next+1), args...)
	if err != nil {
		return shared.ListResult{}, err
	}
	defer rows.Close()
	items := []shared.Record{}
	for rows.Next() {
		rec, err := scanQuestionTagRecord(rows.Scan)
		if err != nil {
			return shared.ListResult{}, err
		}
		items = append(items, rec)
	}
	return shared.ListResult{Items: items, Page: q.Page, Limit: q.Limit, Total: total}, rows.Err()
}

func (s *service) Get(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, id uuid.UUID) (shared.Record, error) {
	admin := canManageAllTags(permissions)
	args := []any{id, tenantID}
	where := "qt.id=$1 AND qt.tenant_id=$2 AND qt.deleted_at IS NULL"
	if !admin {
		where += " AND qt.owner_user_id=$3"
		args = append(args, actorUserID)
	}
	row := s.deps.DB.QueryRow(ctx, `
		SELECT qt.id, qt.tenant_id, qt.code, qt.name, COALESCE(qt.description,''), qt.status, qt.metadata,
		       qt.created_at, qt.updated_at, qt.deleted_at,
		       COALESCE(qt.lecturer_id::text,''), COALESCE(qt.owner_user_id::text,''),
		       COALESCE(l.name,''), COALESCE(u.name,''), COALESCE(u.email,'')
		FROM question_tags qt
		LEFT JOIN lecturers l ON l.id=qt.lecturer_id AND l.deleted_at IS NULL
		LEFT JOIN users u ON u.id=qt.owner_user_id AND u.deleted_at IS NULL
		WHERE `+where, args...)
	return scanQuestionTagRecord(row.Scan)
}

func (s *service) Create(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, req CreateQuestionTagRequest) (shared.Record, error) {
	if err := ValidateCreate(req); err != nil {
		return shared.Record{}, err
	}
	lecturerID, ownerUserID, lecturerName, err := s.resolveTagOwner(ctx, tenantID, actorUserID, permissions, req.LecturerID)
	if err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	rec.ID = uuid.New()
	rec.TenantID = tenantID
	if rec.Status == "" {
		rec.Status = "active"
	}
	if rec.Metadata == nil {
		rec.Metadata = map[string]any{}
	}
	rec.Metadata["lecturer_id"] = lecturerID.String()
	rec.Metadata["owner_user_id"] = ownerUserID.String()
	rec.Metadata["lecturer_name"] = lecturerName
	raw, _ := json.Marshal(rec.Metadata)
	err = s.deps.DB.QueryRow(ctx, `
		INSERT INTO question_tags(id, tenant_id, code, name, description, status, metadata, lecturer_id, owner_user_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at, updated_at`,
		rec.ID, rec.TenantID, rec.Code, rec.Name, rec.Description, rec.Status, raw, lecturerID, ownerUserID).Scan(&rec.CreatedAt, &rec.UpdatedAt)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(rec)
		_ = s.deps.Rabbit.Publish(ctx, "question_tags.created", body)
	}
	return rec, err
}

func (s *service) Update(ctx context.Context, tenantID, id, actorUserID uuid.UUID, permissions []string, req UpdateQuestionTagRequest) (shared.Record, error) {
	if err := ValidateUpdate(req); err != nil {
		return shared.Record{}, err
	}
	if err := s.ensureTagWritable(ctx, tenantID, id, actorUserID, permissions); err != nil {
		return shared.Record{}, err
	}
	lecturerID, ownerUserID, lecturerName, err := s.resolveTagOwner(ctx, tenantID, actorUserID, permissions, req.LecturerID)
	if err != nil {
		return shared.Record{}, err
	}
	meta := req.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	meta["lecturer_id"] = lecturerID.String()
	meta["owner_user_id"] = ownerUserID.String()
	meta["lecturer_name"] = lecturerName
	raw, _ := json.Marshal(meta)
	_, err = s.deps.DB.Exec(ctx, `
		UPDATE question_tags
		SET code=$1, name=$2, description=$3, status=$4, metadata=$5, lecturer_id=$6, owner_user_id=$7, updated_at=now()
		WHERE id=$8 AND tenant_id=$9 AND deleted_at IS NULL`,
		req.Code, req.Name, req.Description, req.Status, raw, lecturerID, ownerUserID, id, tenantID)
	if err != nil {
		return shared.Record{}, err
	}
	out, err := s.Get(ctx, tenantID, actorUserID, permissions, id)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "question_tags.updated", body)
	}
	return out, err
}

func (s *service) Delete(ctx context.Context, tenantID, id, actorUserID uuid.UUID, permissions []string) error {
	if err := s.ensureTagWritable(ctx, tenantID, id, actorUserID, permissions); err != nil {
		return err
	}
	err := s.repo.Delete(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "question_tags.deleted", []byte(id.String()))
	}
	return err
}

func scanQuestionTagRecord(scan func(dest ...any) error) (shared.Record, error) {
	var rec shared.Record
	var raw []byte
	var lecturerID, ownerUserID, lecturerName, ownerName, ownerEmail string
	err := scan(&rec.ID, &rec.TenantID, &rec.Code, &rec.Name, &rec.Description, &rec.Status, &raw, &rec.CreatedAt, &rec.UpdatedAt, &rec.DeletedAt, &lecturerID, &ownerUserID, &lecturerName, &ownerName, &ownerEmail)
	if err != nil {
		return shared.Record{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &rec.Metadata)
	}
	if rec.Metadata == nil {
		rec.Metadata = map[string]any{}
	}
	if lecturerID != "" {
		rec.Metadata["lecturer_id"] = lecturerID
	}
	if ownerUserID != "" {
		rec.Metadata["owner_user_id"] = ownerUserID
	}
	if lecturerName != "" {
		rec.Metadata["lecturer_name"] = lecturerName
	}
	if ownerName != "" {
		rec.Metadata["owner_name"] = ownerName
	}
	if ownerEmail != "" {
		rec.Metadata["owner_email"] = ownerEmail
	}
	return rec, nil
}

func (s *service) resolveTagOwner(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, requestedLecturerID *uuid.UUID) (uuid.UUID, uuid.UUID, string, error) {
	if canManageAllTags(permissions) {
		if requestedLecturerID == nil || *requestedLecturerID == uuid.Nil {
			return uuid.Nil, uuid.Nil, "", errors.New("admin wajib memilih guru pemilik tag")
		}
		var ownerUserID uuid.UUID
		var lecturerName string
		err := s.deps.DB.QueryRow(ctx, `
			SELECT id, user_id, name
			FROM lecturers
			WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL AND status='active'
			LIMIT 1`, *requestedLecturerID, tenantID).Scan(requestedLecturerID, &ownerUserID, &lecturerName)
		if err != nil || ownerUserID == uuid.Nil {
			return uuid.Nil, uuid.Nil, "", errors.New("guru tidak ditemukan atau belum punya akun")
		}
		return *requestedLecturerID, ownerUserID, lecturerName, nil
	}
	var lecturerID, ownerUserID uuid.UUID
	var lecturerName string
	err := s.deps.DB.QueryRow(ctx, `
		SELECT id, user_id, name
		FROM lecturers
		WHERE user_id=$1 AND tenant_id=$2 AND deleted_at IS NULL AND status='active'
		LIMIT 1`, actorUserID, tenantID).Scan(&lecturerID, &ownerUserID, &lecturerName)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", errors.New("akun login ini belum terhubung ke data guru")
	}
	if requestedLecturerID != nil && *requestedLecturerID != uuid.Nil && *requestedLecturerID != lecturerID {
		return uuid.Nil, uuid.Nil, "", errors.New("guru hanya boleh membuat tag untuk dirinya sendiri")
	}
	return lecturerID, ownerUserID, lecturerName, nil
}

func (s *service) ensureTagWritable(ctx context.Context, tenantID, tagID, actorUserID uuid.UUID, permissions []string) error {
	if canManageAllTags(permissions) {
		return nil
	}
	var ownerUserID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `
		SELECT COALESCE(owner_user_id,'00000000-0000-0000-0000-000000000000')
		FROM question_tags
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, tagID, tenantID).Scan(&ownerUserID)
	if err != nil {
		return err
	}
	if ownerUserID != actorUserID {
		return errors.New("tag soal milik guru lain")
	}
	return nil
}

func canManageAllTags(permissions []string) bool {
	for _, permission := range permissions {
		if permission == "*" || permission == "users:read" || permission == "tenants:read" {
			return true
		}
	}
	return false
}
