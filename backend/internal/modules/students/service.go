package students

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/accounts"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
)

type Service interface {
	List(context.Context, uuid.UUID, pagination.Query) (shared.ListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (shared.Record, error)
	Create(context.Context, uuid.UUID, CreateStudentRequest) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateStudentRequest) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
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
func (s *service) Create(ctx context.Context, tenantID uuid.UUID, req CreateStudentRequest) (shared.Record, error) {
	if err := ValidateCreate(req); err != nil {
		return shared.Record{}, err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return shared.Record{}, err
	}
	defer tx.Rollback(ctx)

	out := shared.Record{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		Metadata:    req.Metadata,
	}
	if out.ID == uuid.Nil {
		out.ID = uuid.New()
	}
	if out.TenantID == uuid.Nil {
		out.TenantID = tenantID
	}
	if out.Status == "" {
		out.Status = "active"
	}
	userID, err := accounts.CreateAcademicUser(ctx, tx, s.deps.Config, accounts.AcademicUserRequest{
		TenantID:   tenantID,
		Code:       "STU-" + out.Code,
		Name:       out.Name,
		Email:      strings.TrimSpace(req.Email),
		Password:   req.Password,
		RoleCode:   "STUDENT",
		RoleName:   "Student",
		EntityType: "student",
		EntityID:   out.ID,
	})
	if err != nil {
		return shared.Record{}, err
	}
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	out.Metadata["account_email"] = strings.TrimSpace(req.Email)
	out.Metadata["user_id"] = userID.String()
	raw, _ := json.Marshal(out.Metadata)
	err = tx.QueryRow(ctx, `
		INSERT INTO students(id, tenant_id, code, name, description, status, metadata, user_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING created_at, updated_at`,
		out.ID, out.TenantID, out.Code, out.Name, out.Description, out.Status, raw, userID).Scan(&out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return shared.Record{}, err
	}
	if out.CreatedAt.IsZero() {
		now := time.Now().UTC()
		out.CreatedAt, out.UpdatedAt = now, now
	}
	if err = tx.Commit(ctx); err != nil {
		return shared.Record{}, err
	}
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "students.created", body)
	}
	return out, err
}
func (s *service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateStudentRequest) (shared.Record, error) {
	if err := ValidateUpdate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Update(ctx, tenantID, id, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "students.updated", body)
	}
	return out, err
}
func (s *service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "students.deleted", []byte(id.String()))
	}
	return err
}
