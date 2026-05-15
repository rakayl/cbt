package users

import (
	"time"

	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}
type UpdateUserRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}

type ChangePasswordRequest struct {
	Password string `json:"password" validate:"required,min=8"`
}

type UserListItem struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Code      string         `json:"code"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Status    string         `json:"status"`
	Roles     []string       `json:"roles"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type UserListResult struct {
	Items []UserListItem `json:"items"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Total int64          `json:"total"`
}
