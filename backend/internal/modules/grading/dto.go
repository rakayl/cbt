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

type ReviewSessionView struct {
	SessionID      uuid.UUID      `json:"session_id"`
	ExamID         uuid.UUID      `json:"exam_id"`
	ExamName       string         `json:"exam_name"`
	StudentID      uuid.UUID      `json:"student_id"`
	StudentCode    string         `json:"student_code"`
	StudentName    string         `json:"student_name"`
	Status         string         `json:"status"`
	StartedAt      string         `json:"started_at,omitempty"`
	SubmittedAt    string         `json:"submitted_at,omitempty"`
	Score          float64        `json:"score"`
	MaxScore       float64        `json:"max_score"`
	Percentage     float64        `json:"percentage"`
	Passed         bool           `json:"passed"`
	ManualRequired int            `json:"manual_required"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type ReviewSessionListResult struct {
	Items []ReviewSessionView `json:"items"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
	Total int64               `json:"total"`
}

type ReviewOptionView struct {
	ID       string            `json:"id"`
	Label    string            `json:"label"`
	Text     string            `json:"text"`
	Media    []ReviewMediaView `json:"media,omitempty"`
	Selected bool              `json:"selected"`
	Correct  bool              `json:"correct"`
}

type ReviewMediaView struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	MimeType  string `json:"mime_type,omitempty"`
	FileSize  int64  `json:"file_size,omitempty"`
	SortOrder int    `json:"sort_order,omitempty"`
}

type ReviewQuestionView struct {
	GradingID         uuid.UUID          `json:"grading_id,omitempty"`
	SessionQuestionID uuid.UUID          `json:"session_question_id"`
	QuestionID        uuid.UUID          `json:"question_id"`
	Position          int                `json:"position"`
	QuestionTagName   string             `json:"question_tag_name,omitempty"`
	Text              string             `json:"text"`
	AnswerMode        string             `json:"answer_mode"`
	Media             []ReviewMediaView  `json:"media,omitempty"`
	Options           []ReviewOptionView `json:"options,omitempty"`
	AnswerPayload     map[string]any     `json:"answer_payload,omitempty"`
	SelectedOptionIDs []string           `json:"selected_option_ids,omitempty"`
	CorrectOptionIDs  []string           `json:"correct_option_ids,omitempty"`
	EarnedScore       float64            `json:"earned_score"`
	MaxScore          float64            `json:"max_score"`
	Answered          bool               `json:"answered"`
	IsCorrect         bool               `json:"is_correct"`
	ManualRequired    bool               `json:"manual_required"`
	ManualStatus      string             `json:"manual_status,omitempty"`
	Feedback          string             `json:"feedback,omitempty"`
}

type ReviewSessionDetailResponse struct {
	Session ReviewSessionView    `json:"session"`
	Items   []ReviewQuestionView `json:"items"`
}
