package class_rooms

import (
	"time"

	"github.com/google/uuid"
)

type CreateClassRoomRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	LecturerID  *uuid.UUID     `json:"lecturer_id"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}
type UpdateClassRoomRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	LecturerID  *uuid.UUID     `json:"lecturer_id"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}

type ClassStudentView struct {
	EnrollmentID     uuid.UUID  `json:"enrollment_id"`
	StudentID        uuid.UUID  `json:"student_id"`
	StudentCode      string     `json:"student_code"`
	StudentName      string     `json:"student_name"`
	StudyProgramID   *uuid.UUID `json:"study_program_id,omitempty"`
	StudyProgramName string     `json:"study_program_name,omitempty"`
	Status           string     `json:"status"`
	Active           bool       `json:"active"`
	EnrolledAt       time.Time  `json:"enrolled_at"`
	ExitedAt         *time.Time `json:"exited_at,omitempty"`
}

type ClassStudentListResult struct {
	Items []ClassStudentView `json:"items"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Total int64              `json:"total"`
}
