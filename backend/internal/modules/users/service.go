package users

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/security"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service interface {
	List(context.Context, uuid.UUID, pagination.Query) (UserListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (shared.Record, error)
	Create(context.Context, uuid.UUID, CreateUserRequest) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateUserRequest) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	ChangePassword(context.Context, uuid.UUID, uuid.UUID, ChangePasswordRequest) error
}
type service struct {
	repo Repository
	deps shared.Deps
}

func NewService(repo Repository, deps shared.Deps) Service { return &service{repo: repo, deps: deps} }
func (s *service) List(ctx context.Context, tenantID uuid.UUID, q pagination.Query) (UserListResult, error) {
	where := "u.deleted_at IS NULL AND u.tenant_id=$1"
	args := []any{tenantID}
	if q.Search != "" {
		where += " AND (lower(u.name) LIKE $2 OR lower(u.code) LIKE $2 OR lower(coalesce(u.email,'')) LIKE $2)"
		args = append(args, "%"+strings.ToLower(q.Search)+"%")
	}
	var total int64
	if err := s.deps.DB.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM users u WHERE %s`, where), args...).Scan(&total); err != nil {
		return UserListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	rows, err := s.deps.DB.Query(ctx, fmt.Sprintf(`
		SELECT u.id, u.tenant_id, u.code, u.name, coalesce(u.email,''), u.status, u.metadata, u.created_at, u.updated_at,
		       coalesce(array_agg(r.name ORDER BY r.name) FILTER (WHERE r.id IS NOT NULL), '{}')
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id=u.id AND ur.deleted_at IS NULL
		LEFT JOIN roles r ON r.id=ur.role_id AND r.deleted_at IS NULL
		WHERE %s
		GROUP BY u.id
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return UserListResult{}, err
	}
	defer rows.Close()
	items := []UserListItem{}
	for rows.Next() {
		var item UserListItem
		var raw []byte
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Code, &item.Name, &item.Email, &item.Status, &raw, &item.CreatedAt, &item.UpdatedAt, &item.Roles); err != nil {
			return UserListResult{}, err
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &item.Metadata)
		}
		items = append(items, item)
	}
	return UserListResult{Items: items, Page: q.Page, Limit: q.Limit, Total: total}, rows.Err()
}
func (s *service) Get(ctx context.Context, tenantID, id uuid.UUID) (shared.Record, error) {
	return s.repo.Get(ctx, tenantID, id)
}
func (s *service) Create(ctx context.Context, tenantID uuid.UUID, req CreateUserRequest) (shared.Record, error) {
	if err := ValidateCreate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Create(ctx, tenantID, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "users.created", body)
	}
	return out, err
}
func (s *service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateUserRequest) (shared.Record, error) {
	if err := ValidateUpdate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Update(ctx, tenantID, id, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "users.updated", body)
	}
	return out, err
}
func (s *service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "users.deleted", []byte(id.String()))
	}
	return err
}

func (s *service) ChangePassword(ctx context.Context, tenantID, id uuid.UUID, req ChangePasswordRequest) error {
	if err := validate.Struct(req); err != nil {
		return err
	}
	passwordHash, err := security.HashPassword(req.Password, s.deps.Config.PasswordPepper, s.deps.Config.BcryptCost)
	if err != nil {
		return err
	}
	tag, err := s.deps.DB.Exec(ctx, `UPDATE users SET password_hash=$1, updated_at=now() WHERE id=$2 AND tenant_id=$3 AND deleted_at IS NULL`, passwordHash, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	_, _ = s.deps.DB.Exec(ctx, `UPDATE user_sessions SET revoked_at=now(), status='revoked', updated_at=now() WHERE user_id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, id, tenantID)
	return nil
}
