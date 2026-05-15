package exam_sessions

import (
	"time"

	"github.com/google/uuid"
)

type CreateExamSessionRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}
type UpdateExamSessionRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=80"`
	Name        string         `json:"name" validate:"required,min=2,max=160"`
	Description string         `json:"description" validate:"max=2000"`
	Status      string         `json:"status" validate:"omitempty,oneof=active inactive draft published completed suspended"`
	Metadata    map[string]any `json:"metadata"`
}

type SubmitExamRequest struct {
	ClientSeq int64 `json:"client_seq"`
}

type SubmitExamResponse struct {
	SessionID       uuid.UUID `json:"session_id"`
	Status          string    `json:"status"`
	SubmittedAt     time.Time `json:"submitted_at"`
	Score           float64   `json:"score"`
	MaxScore        float64   `json:"max_score"`
	Percentage      float64   `json:"percentage"`
	PassingGrade    float64   `json:"passing_grade"`
	Passed          bool      `json:"passed"`
	CorrectCount    int       `json:"correct_count"`
	WrongCount      int       `json:"wrong_count"`
	AnsweredCount   int       `json:"answered_count"`
	UnansweredCount int       `json:"unanswered_count"`
}

type StudentSessionView struct {
	SessionID        uuid.UUID      `json:"session_id"`
	ExamID           uuid.UUID      `json:"exam_id"`
	StudentID        uuid.UUID      `json:"student_id"`
	Code             string         `json:"code"`
	ExamName         string         `json:"exam_name"`
	Status           string         `json:"status"`
	StartedAt        *time.Time     `json:"started_at,omitempty"`
	EndsAt           *time.Time     `json:"ends_at,omitempty"`
	SubmittedAt      *time.Time     `json:"submitted_at,omitempty"`
	RemainingSeconds int64          `json:"remaining_seconds"`
	Score            *float64       `json:"score,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type StudentSessionListResult struct {
	Items []StudentSessionView `json:"items"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
	Total int64                `json:"total"`
}

type StudentResultQuestionView struct {
	SessionQuestionID uuid.UUID        `json:"session_question_id"`
	QuestionID        uuid.UUID        `json:"question_id"`
	Position          int              `json:"position"`
	Code              string           `json:"code"`
	Text              string           `json:"text"`
	QuestionType      string           `json:"question_type"`
	AnswerMode        string           `json:"answer_mode"`
	Media             []ExamMediaView  `json:"media,omitempty"`
	Options           []ExamOptionView `json:"options"`
	AnswerPayload     map[string]any   `json:"answer_payload,omitempty"`
	SelectedOptionIDs []string         `json:"selected_option_ids,omitempty"`
	CorrectOptionIDs  []string         `json:"correct_option_ids,omitempty"`
	EarnedScore       float64          `json:"earned_score"`
	MaxScore          float64          `json:"max_score"`
	Answered          bool             `json:"answered"`
	IsCorrect         bool             `json:"is_correct"`
	ManualRequired    bool             `json:"manual_required"`
	ManualStatus      string           `json:"manual_status,omitempty"`
	Feedback          string           `json:"feedback,omitempty"`
	Metadata          map[string]any   `json:"metadata,omitempty"`
}

type StudentResultDetailResponse struct {
	Session   StudentSessionView          `json:"session"`
	Questions []StudentResultQuestionView `json:"questions"`
	Summary   map[string]any              `json:"summary,omitempty"`
}

type ExamOptionView struct {
	ID    uuid.UUID       `json:"id"`
	Label string          `json:"label"`
	Text  string          `json:"text"`
	Media []ExamMediaView `json:"media,omitempty"`
}

type ExamMediaView struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	MimeType  string    `json:"mime_type"`
	FileSize  int64     `json:"file_size"`
	SortOrder int       `json:"sort_order"`
}

type ExamQuestionView struct {
	SessionQuestionID uuid.UUID        `json:"session_question_id"`
	QuestionID        uuid.UUID        `json:"question_id"`
	Position          int              `json:"position"`
	Code              string           `json:"code"`
	Text              string           `json:"text"`
	QuestionType      string           `json:"question_type"`
	AnswerMode        string           `json:"answer_mode"`
	Media             []ExamMediaView  `json:"media,omitempty"`
	Options           []ExamOptionView `json:"options"`
	AnswerPayload     map[string]any   `json:"answer_payload,omitempty"`
}

type ExamQuestionsResponse struct {
	SessionID         uuid.UUID          `json:"session_id"`
	Status            string             `json:"status"`
	StartedAt         time.Time          `json:"started_at"`
	EndsAt            time.Time          `json:"ends_at"`
	ServerTime        time.Time          `json:"server_time"`
	RemainingSeconds  int64              `json:"remaining_seconds"`
	TimerMode         string             `json:"timer_mode"`
	TimerPaused       bool               `json:"timer_paused"`
	RecoveryStatus    string             `json:"recovery_status,omitempty"`
	ReviewRequired    bool               `json:"review_required"`
	ReconnectCount    int                `json:"reconnect_count"`
	TotalPauseSeconds int64              `json:"total_pause_seconds"`
	SuspiciousScore   float64            `json:"suspicious_score"`
	Questions         []ExamQuestionView `json:"questions"`
}
