package class_rooms

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
	List(context.Context, uuid.UUID, pagination.Query, uuid.UUID, []string) (shared.ListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) (shared.Record, error)
	Students(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string, pagination.Query) (ClassStudentListResult, error)
	Create(context.Context, uuid.UUID, uuid.UUID, []string, CreateClassRoomRequest) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string, UpdateClassRoomRequest) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []string) error
}
type service struct {
	repo Repository
	deps shared.Deps
}

func NewService(repo Repository, deps shared.Deps) Service { return &service{repo: repo, deps: deps} }
func (s *service) List(ctx context.Context, tenantID uuid.UUID, q pagination.Query, actorUserID uuid.UUID, permissions []string) (shared.ListResult, error) {
	if canManageAllClassRooms(permissions) {
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
	if err := s.deps.DB.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM class_rooms WHERE %s", where), args...).Scan(&total); err != nil {
		return shared.ListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	rows, err := s.deps.DB.Query(ctx, fmt.Sprintf(`
		SELECT id, tenant_id, code, name, COALESCE(description,''), status, metadata, created_at, updated_at, deleted_at
		FROM class_rooms
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
func (s *service) Get(ctx context.Context, tenantID, id, actorUserID uuid.UUID, permissions []string) (shared.Record, error) {
	if err := s.ensureClassRoomReadable(ctx, tenantID, id, actorUserID, permissions); err != nil {
		return shared.Record{}, err
	}
	return s.repo.Get(ctx, tenantID, id)
}
func (s *service) Students(ctx context.Context, tenantID, classRoomID, actorUserID uuid.UUID, permissions []string, q pagination.Query) (ClassStudentListResult, error) {
	if err := s.ensureClassRoomReadable(ctx, tenantID, classRoomID, actorUserID, permissions); err != nil {
		return ClassStudentListResult{}, err
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 200 {
		q.Limit = 20
	}
	search := "%" + strings.ToLower(q.Search) + "%"
	where := "e.deleted_at IS NULL AND e.tenant_id=$1 AND e.class_room_id=$2"
	args := []any{tenantID, classRoomID}
	nextArg := 3
	if q.Search != "" {
		where += fmt.Sprintf(" AND (lower(s.name) LIKE $%d OR lower(s.code) LIKE $%d)", nextArg, nextArg)
		args = append(args, search)
		nextArg++
	}
	var total int64
	if err := s.deps.DB.QueryRow(ctx, fmt.Sprintf(`
		SELECT count(*)
		FROM enrollment e
		JOIN students s ON s.id=e.student_id AND s.deleted_at IS NULL
		WHERE %s`, where), args...).Scan(&total); err != nil {
		return ClassStudentListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	rows, err := s.deps.DB.Query(ctx, fmt.Sprintf(`
		SELECT e.id, s.id, s.code, s.name, e.study_program_id, COALESCE(sp.name,''), e.status, e.active, e.enrolled_at, e.exited_at
		FROM enrollment e
		JOIN students s ON s.id=e.student_id AND s.deleted_at IS NULL
		LEFT JOIN study_programs sp ON sp.id=e.study_program_id AND sp.deleted_at IS NULL
		WHERE %s
		ORDER BY e.active DESC, e.enrolled_at DESC
		LIMIT $%d OFFSET $%d`, where, nextArg, nextArg+1), args...)
	if err != nil {
		return ClassStudentListResult{}, err
	}
	defer rows.Close()
	items := []ClassStudentView{}
	for rows.Next() {
		var item ClassStudentView
		var studyProgramID *uuid.UUID
		if err := rows.Scan(&item.EnrollmentID, &item.StudentID, &item.StudentCode, &item.StudentName, &studyProgramID, &item.StudyProgramName, &item.Status, &item.Active, &item.EnrolledAt, &item.ExitedAt); err != nil {
			return ClassStudentListResult{}, err
		}
		item.StudyProgramID = studyProgramID
		items = append(items, item)
	}
	return ClassStudentListResult{Items: items, Page: q.Page, Limit: q.Limit, Total: total}, rows.Err()
}
func (s *service) Create(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, req CreateClassRoomRequest) (shared.Record, error) {
	if err := ValidateCreate(req); err != nil {
		return shared.Record{}, err
	}
	lecturerID, ownerUserID, lecturerName, err := s.resolveClassRoomOwner(ctx, tenantID, actorUserID, permissions, req.LecturerID)
	if err != nil {
		return shared.Record{}, err
	}
	out, err := s.insert(ctx, tenantID, lecturerID, ownerUserID, lecturerName, req.Code, req.Name, req.Description, req.Status, req.Metadata)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "class_rooms.created", body)
	}
	return out, err
}
func (s *service) Update(ctx context.Context, tenantID, id, actorUserID uuid.UUID, permissions []string, req UpdateClassRoomRequest) (shared.Record, error) {
	if err := ValidateUpdate(req); err != nil {
		return shared.Record{}, err
	}
	if err := s.ensureClassRoomWritable(ctx, tenantID, id, actorUserID, permissions); err != nil {
		return shared.Record{}, err
	}
	lecturerID, ownerUserID, lecturerName, err := s.resolveClassRoomOwner(ctx, tenantID, actorUserID, permissions, req.LecturerID)
	if err != nil {
		return shared.Record{}, err
	}
	out, err := s.update(ctx, tenantID, id, lecturerID, ownerUserID, lecturerName, req.Code, req.Name, req.Description, req.Status, req.Metadata)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "class_rooms.updated", body)
	}
	return out, err
}
func (s *service) Delete(ctx context.Context, tenantID, actorUserID, id uuid.UUID, permissions []string) error {
	if err := s.ensureClassRoomWritable(ctx, tenantID, id, actorUserID, permissions); err != nil {
		return err
	}
	err := s.repo.Delete(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "class_rooms.deleted", []byte(id.String()))
	}
	return err
}

func (s *service) insert(ctx context.Context, tenantID, lecturerID, ownerUserID uuid.UUID, lecturerName, code, name, description, status string, metadata map[string]any) (shared.Record, error) {
	if status == "" {
		status = "active"
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["lecturer_id"] = lecturerID.String()
	metadata["owner_user_id"] = ownerUserID.String()
	metadata["lecturer_name"] = lecturerName
	raw, _ := json.Marshal(metadata)
	rec := shared.Record{Code: code, Name: name, Description: description, Status: status, Metadata: metadata}
	rec.ID = uuid.New()
	rec.TenantID = tenantID
	err := s.deps.DB.QueryRow(ctx, `
		INSERT INTO class_rooms(id, tenant_id, code, name, description, status, metadata, lecturer_id, owner_user_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at, updated_at`,
		rec.ID, tenantID, code, name, description, status, raw, lecturerID, ownerUserID).Scan(&rec.CreatedAt, &rec.UpdatedAt)
	return rec, err
}

func (s *service) update(ctx context.Context, tenantID, id, lecturerID, ownerUserID uuid.UUID, lecturerName, code, name, description, status string, metadata map[string]any) (shared.Record, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["lecturer_id"] = lecturerID.String()
	metadata["owner_user_id"] = ownerUserID.String()
	metadata["lecturer_name"] = lecturerName
	raw, _ := json.Marshal(metadata)
	rec := shared.Record{Code: code, Name: name, Description: description, Status: status, Metadata: metadata}
	err := s.deps.DB.QueryRow(ctx, `
		UPDATE class_rooms
		SET code=$1, name=$2, description=$3, status=$4, metadata=$5, lecturer_id=$6, owner_user_id=$7, updated_at=now()
		WHERE id=$8 AND tenant_id=$9 AND deleted_at IS NULL
		RETURNING id, tenant_id, created_at, updated_at`,
		code, name, description, status, raw, lecturerID, ownerUserID, id, tenantID).Scan(&rec.ID, &rec.TenantID, &rec.CreatedAt, &rec.UpdatedAt)
	return rec, err
}

func (s *service) resolveClassRoomOwner(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, requestedLecturerID *uuid.UUID) (uuid.UUID, uuid.UUID, string, error) {
	if canManageAllClassRooms(permissions) {
		if requestedLecturerID == nil || *requestedLecturerID == uuid.Nil {
			return uuid.Nil, uuid.Nil, "", errors.New("admin wajib memilih guru pemilik kelas")
		}
		var ownerUserID uuid.UUID
		var lecturerName string
		err := s.deps.DB.QueryRow(ctx, `
			SELECT user_id, name
			FROM lecturers
			WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL AND status='active'
			LIMIT 1`, *requestedLecturerID, tenantID).Scan(&ownerUserID, &lecturerName)
		if err != nil {
			return uuid.Nil, uuid.Nil, "", errors.New("guru tidak ditemukan atau belum punya akun")
		}
		if ownerUserID == uuid.Nil {
			return uuid.Nil, uuid.Nil, "", errors.New("guru belum terhubung dengan akun user")
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
		return uuid.Nil, uuid.Nil, "", errors.New("guru hanya boleh membuat kelas atas nama dirinya sendiri")
	}
	return lecturerID, ownerUserID, lecturerName, nil
}

func (s *service) ensureClassRoomReadable(ctx context.Context, tenantID, classRoomID, actorUserID uuid.UUID, permissions []string) error {
	if canManageAllClassRooms(permissions) {
		var exists bool
		err := s.deps.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM class_rooms WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)`, classRoomID, tenantID).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("class room not found")
		}
		return nil
	}
	var ownerUserID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `
		SELECT COALESCE(owner_user_id,'00000000-0000-0000-0000-000000000000')
		FROM class_rooms
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, classRoomID, tenantID).Scan(&ownerUserID)
	if err != nil {
		return err
	}
	if ownerUserID != actorUserID {
		return errors.New("kelas dimiliki guru lain")
	}
	return nil
}

func (s *service) ensureClassRoomWritable(ctx context.Context, tenantID, classRoomID, actorUserID uuid.UUID, permissions []string) error {
	return s.ensureClassRoomReadable(ctx, tenantID, classRoomID, actorUserID, permissions)
}

func canManageAllClassRooms(permissions []string) bool {
	for _, permission := range permissions {
		if permission == "*" || permission == "users:read" || permission == "tenants:read" {
			return true
		}
	}
	return false
}
