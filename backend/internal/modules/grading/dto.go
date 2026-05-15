package grading

import "github.com/google/uuid"

type CreateGradingRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}
type UpdateGradingRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}

type ManualScoreRequest struct {
	EarnedScore float64 `json:"earned_score" validate:"gte=0"`
	Feedback    string  `json:"feedback" validate:"max=5000"`
	Status      string  `json:"status" validate:"omitempty,oneof=reviewed needs_revision"`
}

type GradeItemView struct {
	ID                uuid.UUID      `json:"id"`
	ExamSessionID     uuid.UUID      `json:"exam_session_id"`
	SessionQuestionID uuid.UUID      `json:"session_question_id"`
	QuestionID        uuid.UUID      `json:"question_id"`
	QuestionText      string         `json:"question_text,omitempty"`
	AnswerMode        string         `json:"answer_mode"`
	SelectedOptionIDs []string       `json:"selected_option_ids,omitempty"`
	CorrectOptionIDs  []string       `json:"correct_option_ids,omitempty"`
	EarnedScore       float64        `json:"earned_score"`
	MaxScore          float64        `json:"max_score"`
	Answered          bool           `json:"answered"`
	IsCorrect         bool           `json:"is_correct"`
	ManualRequired    bool           `json:"manual_required"`
	ManualStatus      string         `json:"manual_status,omitempty"`
	Feedback          string         `json:"feedback,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

type SessionGradesResponse struct {
	ExamSessionID uuid.UUID       `json:"exam_session_id"`
	Status        string          `json:"status"`
	Score         float64         `json:"score"`
	MaxScore      float64         `json:"max_score"`
	Percentage    float64         `json:"percentage"`
	Passed        bool            `json:"passed"`
	Items         []GradeItemView `json:"items"`
}
