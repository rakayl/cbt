package roles

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
)

type Service interface {
	List(context.Context, uuid.UUID, pagination.Query) (shared.ListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (shared.Record, error)
	Create(context.Context, uuid.UUID, CreateRoleRequest) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateRoleRequest) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	Permissions(context.Context, uuid.UUID, uuid.UUID) (RolePermissionsResult, error)
	SetPermissions(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, SetRolePermissionsRequest) (RolePermissionsResult, error)
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
func (s *service) Create(ctx context.Context, tenantID uuid.UUID, req CreateRoleRequest) (shared.Record, error) {
	if err := ValidateCreate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Create(ctx, tenantID, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "roles.created", body)
	}
	return out, err
}
func (s *service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateRoleRequest) (shared.Record, error) {
	if err := ValidateUpdate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Update(ctx, tenantID, id, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "roles.updated", body)
	}
	return out, err
}
func (s *service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "roles.deleted", []byte(id.String()))
	}
	return err
}

func (s *service) Permissions(ctx context.Context, tenantID, roleID uuid.UUID) (RolePermissionsResult, error) {
	var out RolePermissionsResult
	if err := s.deps.DB.QueryRow(ctx, `
		SELECT id, code, name
		FROM roles
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		roleID, tenantID).Scan(&out.RoleID, &out.RoleCode, &out.RoleName); err != nil {
		return out, err
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT p.id, p.code, p.name, COALESCE(p.description,''), p.status, p.metadata,
		       EXISTS(
		         SELECT 1
		         FROM role_permissions rp
		         WHERE rp.role_id=$1 AND rp.permission_id=p.id AND rp.deleted_at IS NULL
		       ) AS assigned
		FROM permissions p
		WHERE p.tenant_id=$2 AND p.deleted_at IS NULL
		ORDER BY p.code ASC`,
		roleID, tenantID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.Permissions = []RolePermissionView{}
	for rows.Next() {
		var item RolePermissionView
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Status, &raw, &item.Assigned); err != nil {
			return out, err
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &item.Metadata)
		}
		out.Permissions = append(out.Permissions, item)
	}
	return out, rows.Err()
}

func (s *service) SetPermissions(ctx context.Context, tenantID, roleID, actorUserID uuid.UUID, req SetRolePermissionsRequest) (RolePermissionsResult, error) {
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return RolePermissionsResult{}, err
	}
	defer tx.Rollback(ctx)

	var roleCode, roleName string
	if err := tx.QueryRow(ctx, `
		SELECT code, name
		FROM roles
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		roleID, tenantID).Scan(&roleCode, &roleName); err != nil {
		return RolePermissionsResult{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE role_permissions
		SET deleted_at=now(), updated_at=now(), status='inactive'
		WHERE role_id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		roleID, tenantID)
	if err != nil {
		return RolePermissionsResult{}, err
	}
	seen := map[uuid.UUID]bool{}
	for _, permissionID := range req.PermissionIDs {
		if permissionID == uuid.Nil || seen[permissionID] {
			continue
		}
		seen[permissionID] = true
		var permissionCode, permissionName string
		if err := tx.QueryRow(ctx, `
			SELECT code, name
			FROM permissions
			WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
			permissionID, tenantID).Scan(&permissionCode, &permissionName); err != nil {
			return RolePermissionsResult{}, fmt.Errorf("permission tidak ditemukan: %s", permissionID)
		}
		tag, err := tx.Exec(ctx, `
			UPDATE role_permissions
			SET deleted_at=NULL, status='active', updated_at=now(), code=$1, name=$2
			WHERE role_id=$3 AND permission_id=$4 AND tenant_id=$5`,
			roleCode+"_"+permissionCode, roleName+" "+permissionName, roleID, permissionID, tenantID)
		if err != nil {
			return RolePermissionsResult{}, err
		}
		if tag.RowsAffected() == 0 {
			meta, _ := json.Marshal(map[string]any{
				"managed_by":        actorUserID.String(),
				"managed_via":       "rbac_permission_management",
				"permission_code":   permissionCode,
				"permission_name":   permissionName,
				"role_code":         roleCode,
				"role_name":         roleName,
				"assignment_source": "admin_ui",
			})
			if _, err := tx.Exec(ctx, `
				INSERT INTO role_permissions(id,tenant_id,code,name,role_id,permission_id,status,metadata)
				VALUES($1,$2,$3,$4,$5,$6,'active',$7)`,
				uuid.New(), tenantID, roleCode+"_"+permissionCode, roleName+" "+permissionName, roleID, permissionID, meta); err != nil {
				return RolePermissionsResult{}, err
			}
		}
	}
	auditMeta, _ := json.Marshal(map[string]any{
		"role_id":        roleID.String(),
		"role_code":      roleCode,
		"permission_ids": req.PermissionIDs,
	})
	_, _ = tx.Exec(ctx, `
		INSERT INTO audit_logs(id,tenant_id,code,name,status,actor_user_id,entity_type,entity_id,event_type,metadata)
		VALUES($1,$2,$3,$4,'active',$5,'roles',$6,'role.permissions.update',$7)`,
		uuid.New(), tenantID, "RBAC-"+roleID.String()[:8], "Role permissions updated", actorUserID, roleID, auditMeta)
	if err := tx.Commit(ctx); err != nil {
		return RolePermissionsResult{}, err
	}
	if s.deps.Rabbit != nil {
		body, _ := json.Marshal(map[string]any{"role_id": roleID, "permission_ids": req.PermissionIDs})
		_ = s.deps.Rabbit.Publish(ctx, "roles.permissions.updated", body)
	}
	return s.Permissions(ctx, tenantID, roleID)
}
