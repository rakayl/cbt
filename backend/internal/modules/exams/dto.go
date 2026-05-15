package exams

import (
	"time"

	"github.com/google/uuid"
)

type CreateExamRequest struct {
	Code            string         `json:"code" validate:"required,min=2,max=80"`
	Name            string         `json:"name" validate:"required,min=2,max=160"`
	Description     string         `json:"description" validate:"max=2000"`
	Status          string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata        map[string]any `json:"metadata"`
	CourseClassID   *uuid.UUID     `json:"course_class_id"`
	DurationMinutes int            `json:"duration_minutes" validate:"omitempty,gte=1,lte=1440"`
	PassingGrade    float64        `json:"passing_grade" validate:"omitempty,gte=0,lte=100"`
	ExamToken       string         `json:"exam_token" validate:"omitempty,min=4,max=64"`
	RandomQuestion  *bool          `json:"random_question"`
	RandomOption    *bool          `json:"random_option"`
	QuestionCount   int            `json:"question_count" validate:"omitempty,gte=1,lte=500"`
	MaxAttempt      int            `json:"max_attempt" validate:"omitempty,gte=1,lte=20"`
	Instruction     string         `json:"instruction" validate:"max=5000"`
}
type UpdateExamRequest struct {
	Code            string         `json:"code" validate:"required,min=2,max=80"`
	Name            string         `json:"name" validate:"required,min=2,max=160"`
	Description     string         `json:"description" validate:"max=2000"`
	Status          string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata        map[string]any `json:"metadata"`
	CourseClassID   *uuid.UUID     `json:"course_class_id"`
	DurationMinutes int            `json:"duration_minutes" validate:"omitempty,gte=1,lte=1440"`
	PassingGrade    float64        `json:"passing_grade" validate:"omitempty,gte=0,lte=100"`
	ExamToken       string         `json:"exam_token" validate:"omitempty,min=4,max=64"`
	RandomQuestion  *bool          `json:"random_question"`
	RandomOption    *bool          `json:"random_option"`
	QuestionCount   int            `json:"question_count" validate:"omitempty,gte=1,lte=500"`
	MaxAttempt      int            `json:"max_attempt" validate:"omitempty,gte=1,lte=20"`
	Instruction     string         `json:"instruction" validate:"max=5000"`
}

type PublishExamRequest struct {
	DurationMinutes int                   `json:"duration_minutes" validate:"omitempty,gte=1,lte=1440"`
	PassingGrade    float64               `json:"passing_grade" validate:"omitempty,gte=0,lte=100"`
	ExamToken       string                `json:"exam_token" validate:"omitempty,min=4,max=64"`
	Instruction     string                `json:"instruction" validate:"max=5000"`
	RandomQuestion  *bool                 `json:"random_question"`
	RandomOption    *bool                 `json:"random_option"`
	QuestionCount   int                   `json:"question_count" validate:"omitempty,gte=1,lte=500"`
	QuestionPools   []QuestionPoolRequest `json:"question_pools" validate:"omitempty,dive"`
	MaxAttempt      int                   `json:"max_attempt" validate:"omitempty,gte=1,lte=20"`
	CourseClassID   *uuid.UUID            `json:"course_class_id"`
	Metadata        map[string]any        `json:"metadata"`
}

type QuestionPoolRequest struct {
	QuestionTagID uuid.UUID `json:"question_tag_id" validate:"required"`
	QuestionCount int       `json:"question_count" validate:"required,gte=1,lte=500"`
}

type PublishExamResponse struct {
	ID              uuid.UUID          `json:"id"`
	ExamToken       string             `json:"exam_token"`
	Status          string             `json:"status"`
	PublishedAt     string             `json:"published_at"`
	DurationMinutes int                `json:"duration_minutes"`
	PassingGrade    float64            `json:"passing_grade"`
	QuestionCount   int                `json:"question_count"`
	QuestionPools   []QuestionPoolView `json:"question_pools,omitempty"`
	MaxAttempt      int                `json:"max_attempt"`
}

type ExamRevisionResponse struct {
	SourceExamID   uuid.UUID `json:"source_exam_id"`
	RevisionExamID uuid.UUID `json:"revision_exam_id"`
	RevisionNumber int       `json:"revision_number"`
	Status         string    `json:"status"`
}

type ExamRevisionView struct {
	ID             uuid.UUID      `json:"id"`
	Code           string         `json:"code"`
	Name           string         `json:"name"`
	Status         string         `json:"status"`
	RevisionNumber int            `json:"revision_number"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

type ExamRevisionListResult struct {
	Items []ExamRevisionView `json:"items"`
	Total int                `json:"total"`
}

type QuestionPoolView struct {
	QuestionTagID   uuid.UUID `json:"question_tag_id"`
	QuestionTagName string    `json:"question_tag_name"`
	QuestionCount   int       `json:"question_count"`
}

type InviteStudentsRequest struct {
	StudentIDs []uuid.UUID `json:"student_ids" validate:"required,min=1,dive,required"`
}

type ExamInviteView struct {
	ID             uuid.UUID `json:"id"`
	ExamID         uuid.UUID `json:"exam_id"`
	StudentID      uuid.UUID `json:"student_id"`
	StudentCode    string    `json:"student_code"`
	StudentName    string    `json:"student_name"`
	InvitationCode string    `json:"invitation_code"`
	Status         string    `json:"status"`
}

type ShareCodeResponse struct {
	ExamID uuid.UUID `json:"exam_id"`
	Code   string    `json:"code"`
}

type JoinByCodeRequest struct {
	Code string `json:"code" validate:"required,min=4,max=64"`
}

type JoinByCodeResponse struct {
	ExamID          uuid.UUID `json:"exam_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	ExamToken       string    `json:"exam_token"`
	InvitationCode  string    `json:"invitation_code,omitempty"`
	StudentID       uuid.UUID `json:"student_id"`
	Invited         bool      `json:"invited"`
	DurationMinutes int       `json:"duration_minutes"`
}

type StudentExamView struct {
	ExamID          uuid.UUID      `json:"exam_id"`
	StudentID       uuid.UUID      `json:"student_id"`
	Code            string         `json:"code"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	Status          string         `json:"status"`
	ExamToken       string         `json:"exam_token,omitempty"`
	InvitationCode  string         `json:"invitation_code,omitempty"`
	Invited         bool           `json:"invited"`
	DurationMinutes int            `json:"duration_minutes"`
	PassingGrade    float64        `json:"passing_grade"`
	QuestionCount   int            `json:"question_count"`
	MaxAttempt      int            `json:"max_attempt"`
	Instruction     string         `json:"instruction,omitempty"`
	SessionID       *uuid.UUID     `json:"session_id,omitempty"`
	SessionStatus   string         `json:"session_status,omitempty"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	EndsAt          *time.Time     `json:"ends_at,omitempty"`
	SubmittedAt     *time.Time     `json:"submitted_at,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}
