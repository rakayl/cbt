package enrollment

import (
	"time"

	"github.com/google/uuid"
)

type CreateEnrollmentRequest struct {
	StudentID      uuid.UUID      `json:"student_id" validate:"required"`
	ClassRoomID    uuid.UUID      `json:"class_room_id" validate:"required"`
	StudyProgramID uuid.UUID      `json:"study_program_id"`
	Description    string         `json:"description" validate:"max=2000"`
	Status         string         `json:"status" validate:"omitempty,oneof=active inactive completed suspended"`
	Metadata       map[string]any `json:"metadata"`
}

type UpdateEnrollmentRequest struct {
	StudentID      uuid.UUID      `json:"student_id" validate:"required"`
	ClassRoomID    uuid.UUID      `json:"class_room_id" validate:"required"`
	StudyProgramID uuid.UUID      `json:"study_program_id"`
	Description    string         `json:"description" validate:"max=2000"`
	Status         string         `json:"status" validate:"omitempty,oneof=active inactive completed suspended"`
	Active         bool           `json:"active"`
	ExitedAt       *time.Time     `json:"exited_at"`
	Metadata       map[string]any `json:"metadata"`
}

type EnrollmentFilters struct {
	StudentID   uuid.UUID
	ClassRoomID uuid.UUID
	Search      string
	Page        int
	Limit       int
}

type EnrollmentView struct {
	ID               uuid.UUID      `json:"id"`
	TenantID         uuid.UUID      `json:"tenant_id"`
	Code             string         `json:"code"`
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	Status           string         `json:"status"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	StudentID        uuid.UUID      `json:"student_id"`
	StudentCode      string         `json:"student_code"`
	StudentName      string         `json:"student_name"`
	ClassRoomID      uuid.UUID      `json:"class_room_id"`
	ClassRoomCode    string         `json:"class_room_code"`
	ClassRoomName    string         `json:"class_room_name"`
	StudyProgramID   uuid.UUID      `json:"study_program_id,omitempty"`
	StudyProgramName string         `json:"study_program_name,omitempty"`
	EnrolledAt       time.Time      `json:"enrolled_at"`
	ExitedAt         *time.Time     `json:"exited_at,omitempty"`
	Active           bool           `json:"active"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type EnrollmentListResult struct {
	Items []EnrollmentView `json:"items"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
	Total int64            `json:"total"`
}
