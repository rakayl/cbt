package accounts

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AcademicUserRequest struct {
	TenantID   uuid.UUID
	Code       string
	Name       string
	Email      string
	Password   string
	RoleCode   string
	RoleName   string
	EntityType string
	EntityID   uuid.UUID
}

func CreateAcademicUser(ctx context.Context, tx pgx.Tx, cfg config.Config, req AcademicUserRequest) (uuid.UUID, error) {
	roleID, err := ensureRole(ctx, tx, req.TenantID, req.RoleCode, req.RoleName)
	if err != nil {
		return uuid.Nil, err
	}
	userID := uuid.New()
	passwordHash, err := security.HashPassword(req.Password, cfg.PasswordPepper, cfg.BcryptCost)
	if err != nil {
		return uuid.Nil, err
	}
	metadata, _ := json.Marshal(map[string]any{
		"created_from": "academic_management",
		"entity_type":  req.EntityType,
		"entity_id":    req.EntityID.String(),
		"role":         req.RoleCode,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO users(id,tenant_id,code,name,email,password_hash,status,metadata)
		VALUES($1,$2,$3,$4,$5,$6,'active',$7)`,
		userID, req.TenantID, maxLen(req.Code, 80), req.Name, strings.TrimSpace(req.Email), passwordHash, metadata)
	if err != nil {
		return uuid.Nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_roles(id,tenant_id,code,name,user_id,role_id,status,metadata)
		VALUES($1,$2,$3,$4,$5,$6,'active',$7)
		ON CONFLICT (user_id, role_id) WHERE deleted_at IS NULL DO NOTHING`,
		uuid.New(), req.TenantID, req.RoleCode, req.RoleName, userID, roleID, metadata)
	if err != nil {
		return uuid.Nil, err
	}
	if err = ensureRolePermissions(ctx, tx, req.TenantID, roleID, req.RoleCode); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func maxLen(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func ensureRolePermissions(ctx context.Context, tx pgx.Tx, tenantID, roleID uuid.UUID, roleCode string) error {
	codes := []string{}
	switch roleCode {
	case "LECTURER":
		codes = []string{"dashboard:read", "students:read", "class.rooms:read", "class.rooms:write", "enrollment:read", "courses:read", "question.banks:read", "question.banks:write", "question.tags:read", "question.tags:write", "questions:read", "questions:write", "exams:read", "exams:write", "exams:invite", "exam.sessions:read", "analytics:read", "reports:read", "proctoring:read"}
	case "STUDENT":
		codes = []string{"dashboard:read", "exams:join", "exam.sessions:read", "exam.sessions:write", "proctoring:write"}
	default:
		return nil
	}
	for _, code := range codes {
		permissionID, err := ensurePermission(ctx, tx, tenantID, code)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO role_permissions(id,tenant_id,code,name,role_id,permission_id,status,metadata)
			VALUES($1,$2,$3,$3,$4,$5,'active','{"seed":true}')
			ON CONFLICT (role_id, permission_id) WHERE deleted_at IS NULL DO NOTHING`,
			uuid.New(), tenantID, roleCode+"_"+code, roleID, permissionID)
		if err != nil {
			return err
		}
	}
	return nil
}

func ensurePermission(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, code string) (uuid.UUID, error) {
	var permissionID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id FROM permissions
		WHERE tenant_id=$1 AND code=$2 AND deleted_at IS NULL
		LIMIT 1`, tenantID, code).Scan(&permissionID)
	if err == nil {
		return permissionID, nil
	}
	parts := strings.SplitN(code, ":", 2)
	resource, action := code, "access"
	if len(parts) == 2 {
		resource, action = parts[0], parts[1]
	}
	permissionID = uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO permissions(id,tenant_id,code,name,resource,action,status,metadata)
		VALUES($1,$2,$3,$3,$4,$5,'active','{"seed":true}')`,
		permissionID, tenantID, code, resource, action)
	if err != nil {
		return uuid.Nil, err
	}
	return permissionID, nil
}

func ensureRole(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, code, name string) (uuid.UUID, error) {
	var roleID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM roles
		WHERE tenant_id=$1 AND code=$2 AND deleted_at IS NULL
		LIMIT 1`, tenantID, code).Scan(&roleID)
	if err == nil {
		return roleID, nil
	}
	roleID = uuid.New()
	metadata, _ := json.Marshal(map[string]any{"system": true, "academic_role": true})
	_, err = tx.Exec(ctx, `
		INSERT INTO roles(id,tenant_id,code,name,status,metadata)
		VALUES($1,$2,$3,$4,'active',$5)`,
		roleID, tenantID, code, name, metadata)
	if err != nil {
		return uuid.Nil, err
	}
	return roleID, nil
}
