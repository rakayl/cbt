package roles

import "github.com/google/uuid"

type CreateRoleRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}
type UpdateRoleRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}

type SetRolePermissionsRequest struct {
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}

type RolePermissionView struct {
	ID          uuid.UUID      `json:"id"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Assigned    bool           `json:"assigned"`
}

type RolePermissionsResult struct {
	RoleID      uuid.UUID            `json:"role_id"`
	RoleCode    string               `json:"role_code"`
	RoleName    string               `json:"role_name"`
	Permissions []RolePermissionView `json:"permissions"`
}
