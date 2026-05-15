package enrollment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
)

type Service interface {
	List(context.Context, uuid.UUID, pagination.Query, EnrollmentFilters) (EnrollmentListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (EnrollmentView, error)
	StudentHistory(context.Context, uuid.UUID, uuid.UUID, pagination.Query) (EnrollmentListResult, error)
	MyClasses(context.Context, uuid.UUID, uuid.UUID, pagination.Query) (EnrollmentListResult, error)
	Create(context.Context, uuid.UUID, CreateEnrollmentRequest) (EnrollmentView, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateEnrollmentRequest) (EnrollmentView, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}

type service struct {
	repo Repository
	deps shared.Deps
}

func NewService(repo Repository, deps shared.Deps) Service { return &service{repo: repo, deps: deps} }

func (s *service) List(ctx context.Context, tenantID uuid.UUID, q pagination.Query, filters EnrollmentFilters) (EnrollmentListResult, error) {
	filters.Page, filters.Limit, filters.Search = q.Page, q.Limit, q.Search
	return s.list(ctx, tenantID, filters)
}

func (s *service) StudentHistory(ctx context.Context, tenantID, studentID uuid.UUID, q pagination.Query) (EnrollmentListResult, error) {
	return s.list(ctx, tenantID, EnrollmentFilters{StudentID: studentID, Search: q.Search, Page: q.Page, Limit: q.Limit})
}

func (s *service) MyClasses(ctx context.Context, tenantID, userID uuid.UUID, q pagination.Query) (EnrollmentListResult, error) {
	var studentID uuid.UUID
	if err := s.deps.DB.QueryRow(ctx, `SELECT id FROM students WHERE tenant_id=$1 AND user_id=$2 AND deleted_at IS NULL LIMIT 1`, tenantID, userID).Scan(&studentID); err != nil {
		return EnrollmentListResult{}, err
	}
	return s.StudentHistory(ctx, tenantID, studentID, q)
}

func (s *service) Get(ctx context.Context, tenantID, id uuid.UUID) (EnrollmentView, error) {
	rows, err := s.deps.DB.Query(ctx, enrollmentSelectSQL()+` AND e.tenant_id=$1 AND e.id=$2 LIMIT 1`, tenantID, id)
	if err != nil {
		return EnrollmentView{}, err
	}
	defer rows.Close()
	if rows.Next() {
		return scanEnrollment(rows)
	}
	return EnrollmentView{}, rows.Err()
}

func (s *service) Create(ctx context.Context, tenantID uuid.UUID, req CreateEnrollmentRequest) (EnrollmentView, error) {
	if err := ValidateCreate(req); err != nil {
		return EnrollmentView{}, err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return EnrollmentView{}, err
	}
	defer tx.Rollback(ctx)

	var studentName, className, studyProgramName string
	var studentCode, classCode string
	if err = tx.QueryRow(ctx, `SELECT code, name FROM students WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, req.StudentID, tenantID).Scan(&studentCode, &studentName); err != nil {
		return EnrollmentView{}, err
	}
	if err = tx.QueryRow(ctx, `SELECT code, name FROM class_rooms WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, req.ClassRoomID, tenantID).Scan(&classCode, &className); err != nil {
		return EnrollmentView{}, err
	}
	if req.StudyProgramID != uuid.Nil {
		if err = tx.QueryRow(ctx, `SELECT name FROM study_programs WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, req.StudyProgramID, tenantID).Scan(&studyProgramName); err != nil {
			return EnrollmentView{}, err
		}
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	active := status == "active"
	if active {
		_, err = tx.Exec(ctx, `
			UPDATE enrollment
			SET active=false, status='completed', exited_at=now(), updated_at=now()
			WHERE tenant_id=$1 AND student_id=$2 AND active=true AND deleted_at IS NULL`,
			tenantID, req.StudentID)
		if err != nil {
			return EnrollmentView{}, err
		}
	}
	id := uuid.New()
	code := "ENR-" + id.String()[:8]
	name := strings.TrimSpace(studentName + " -> " + className)
	meta := req.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	meta["student_code"] = studentCode
	meta["class_room_code"] = classCode
	if studyProgramName != "" {
		meta["study_program_name"] = studyProgramName
	}
	raw, _ := json.Marshal(meta)
	_, err = tx.Exec(ctx, `
		INSERT INTO enrollment(id,tenant_id,code,name,description,status,metadata,student_id,class_room_id,study_program_id,enrolled_at,active)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10::text,'00000000-0000-0000-0000-000000000000')::uuid,now(),$11)`,
		id, tenantID, code, name, req.Description, status, raw, req.StudentID, req.ClassRoomID, req.StudyProgramID, active)
	if err != nil {
		return EnrollmentView{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return EnrollmentView{}, err
	}
	out, err := s.Get(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "enrollment.created", body)
	}
	return out, err
}

func (s *service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateEnrollmentRequest) (EnrollmentView, error) {
	if err := ValidateUpdate(req); err != nil {
		return EnrollmentView{}, err
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	raw, _ := json.Marshal(req.Metadata)
	_, err := s.deps.DB.Exec(ctx, `
		UPDATE enrollment
		SET student_id=$1, class_room_id=$2, study_program_id=NULLIF($3::text,'00000000-0000-0000-0000-000000000000')::uuid,
		    description=$4, status=$5, active=$6, exited_at=$7, metadata=$8, updated_at=now()
		WHERE id=$9 AND tenant_id=$10 AND deleted_at IS NULL`,
		req.StudentID, req.ClassRoomID, req.StudyProgramID, req.Description, status, req.Active, req.ExitedAt, raw, id, tenantID)
	if err != nil {
		return EnrollmentView{}, err
	}
	out, err := s.Get(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "enrollment.updated", body)
	}
	return out, err
}

func (s *service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.deps.DB.Exec(ctx, `
		UPDATE enrollment
		SET deleted_at=now(), active=false, updated_at=now()
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, id, tenantID)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "enrollment.deleted", []byte(id.String()))
	}
	return err
}

func (s *service) list(ctx context.Context, tenantID uuid.UUID, filters EnrollmentFilters) (EnrollmentListResult, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 || filters.Limit > 200 {
		filters.Limit = 20
	}
	where, args := enrollmentWhere(tenantID, filters)
	countSQL := fmt.Sprintf(`SELECT count(*) FROM enrollment e JOIN students s ON s.id=e.student_id JOIN class_rooms cr ON cr.id=e.class_room_id WHERE %s`, where)
	var total int64
	if err := s.deps.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return EnrollmentListResult{}, err
	}
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
	rows, err := s.deps.DB.Query(ctx, enrollmentSelectSQL()+` AND `+where+fmt.Sprintf(` ORDER BY e.enrolled_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return EnrollmentListResult{}, err
	}
	defer rows.Close()
	items := []EnrollmentView{}
	for rows.Next() {
		item, err := scanEnrollment(rows)
		if err != nil {
			return EnrollmentListResult{}, err
		}
		items = append(items, item)
	}
	return EnrollmentListResult{Items: items, Page: filters.Page, Limit: filters.Limit, Total: total}, rows.Err()
}

func enrollmentWhere(tenantID uuid.UUID, filters EnrollmentFilters) (string, []any) {
	where := "e.deleted_at IS NULL AND e.tenant_id=$1"
	args := []any{tenantID}
	next := 2
	if filters.StudentID != uuid.Nil {
		where += fmt.Sprintf(" AND e.student_id=$%d", next)
		args = append(args, filters.StudentID)
		next++
	}
	if filters.ClassRoomID != uuid.Nil {
		where += fmt.Sprintf(" AND e.class_room_id=$%d", next)
		args = append(args, filters.ClassRoomID)
		next++
	}
	if filters.Search != "" {
		where += fmt.Sprintf(" AND (lower(s.name) LIKE $%d OR lower(s.code) LIKE $%d OR lower(cr.name) LIKE $%d OR lower(cr.code) LIKE $%d)", next, next, next, next)
		args = append(args, "%"+strings.ToLower(filters.Search)+"%")
	}
	return where, args
}

func enrollmentSelectSQL() string {
	return `
		SELECT e.id, e.tenant_id, e.code, e.name, coalesce(e.description,''), e.status, e.metadata,
		       e.student_id, s.code, s.name,
		       e.class_room_id, cr.code, cr.name,
		       coalesce(e.study_program_id,'00000000-0000-0000-0000-000000000000'), coalesce(sp.name,''),
		       e.enrolled_at, e.exited_at, e.active, e.created_at, e.updated_at
		FROM enrollment e
		JOIN students s ON s.id=e.student_id AND s.deleted_at IS NULL
		JOIN class_rooms cr ON cr.id=e.class_room_id AND cr.deleted_at IS NULL
		LEFT JOIN study_programs sp ON sp.id=e.study_program_id AND sp.deleted_at IS NULL
		WHERE e.deleted_at IS NULL`
}

type enrollmentScanner interface {
	Scan(dest ...any) error
}

func scanEnrollment(row enrollmentScanner) (EnrollmentView, error) {
	var out EnrollmentView
	var raw []byte
	err := row.Scan(&out.ID, &out.TenantID, &out.Code, &out.Name, &out.Description, &out.Status, &raw,
		&out.StudentID, &out.StudentCode, &out.StudentName,
		&out.ClassRoomID, &out.ClassRoomCode, &out.ClassRoomName,
		&out.StudyProgramID, &out.StudyProgramName,
		&out.EnrolledAt, &out.ExitedAt, &out.Active, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return EnrollmentView{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out.Metadata)
	}
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	return out, nil
}
