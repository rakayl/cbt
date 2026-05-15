package questions

import "github.com/google/uuid"

type QuestionOptionInput struct {
	ID        *uuid.UUID `json:"id"`
	Label     string     `json:"label" validate:"required,min=1,max=8"`
	Text      string     `json:"text" validate:"max=2000"`
	IsCorrect bool       `json:"is_correct"`
}

type CreateQuestionRequest struct {
	Code           string                `json:"code" validate:"required,min=2,max=80"`
	Name           string                `json:"name" validate:"required,min=2,max=160"`
	QuestionBankID uuid.UUID             `json:"question_bank_id" validate:"required"`
	LecturerID     *uuid.UUID            `json:"lecturer_id"`
	QuestionText   string                `json:"question_text" validate:"max=10000"`
	AnswerMode     string                `json:"answer_mode" validate:"required,oneof=single multiple"`
	Difficulty     string                `json:"difficulty" validate:"omitempty,oneof=easy medium hard"`
	Score          float64               `json:"score" validate:"gte=0"`
	Explanation    string                `json:"explanation" validate:"max=2000"`
	Status         string                `json:"status" validate:"omitempty,oneof=active inactive draft archived retired published completed suspended"`
	Options        []QuestionOptionInput `json:"options" validate:"required,min=2,dive"`
	TagIDs         []uuid.UUID           `json:"tag_ids"`
	Metadata       map[string]any        `json:"metadata"`
}

type UpdateQuestionRequest struct {
	Code           string                `json:"code" validate:"required,min=2,max=80"`
	Name           string                `json:"name" validate:"required,min=2,max=160"`
	QuestionBankID uuid.UUID             `json:"question_bank_id" validate:"required"`
	LecturerID     *uuid.UUID            `json:"lecturer_id"`
	QuestionText   string                `json:"question_text" validate:"max=10000"`
	AnswerMode     string                `json:"answer_mode" validate:"required,oneof=single multiple"`
	Difficulty     string                `json:"difficulty" validate:"omitempty,oneof=easy medium hard"`
	Score          float64               `json:"score" validate:"gte=0"`
	Explanation    string                `json:"explanation" validate:"max=2000"`
	Status         string                `json:"status" validate:"omitempty,oneof=active inactive draft archived retired published completed suspended"`
	Options        []QuestionOptionInput `json:"options" validate:"required,min=2,dive"`
	TagIDs         []uuid.UUID           `json:"tag_ids"`
	Metadata       map[string]any        `json:"metadata"`
}

type QuestionOptionView struct {
	ID        uuid.UUID           `json:"id"`
	Label     string              `json:"label"`
	Text      string              `json:"text"`
	IsCorrect bool                `json:"is_correct"`
	SortOrder int                 `json:"sort_order"`
	Media     []QuestionMediaView `json:"media,omitempty"`
	Metadata  map[string]any      `json:"metadata,omitempty"`
}

type QuestionMediaView struct {
	ID               uuid.UUID  `json:"id"`
	QuestionID       uuid.UUID  `json:"question_id"`
	OptionID         *uuid.UUID `json:"option_id,omitempty"`
	MediaType        string     `json:"media_type"`
	UsageType        string     `json:"usage_type"`
	ObjectKey        string     `json:"object_key"`
	OriginalFilename string     `json:"original_filename,omitempty"`
	MimeType         string     `json:"mime_type"`
	FileSize         int64      `json:"file_size"`
	Width            *int       `json:"width,omitempty"`
	Height           *int       `json:"height,omitempty"`
	SortOrder        int        `json:"sort_order"`
	URL              string     `json:"url"`
}

type QuestionTagView struct {
	ID           uuid.UUID  `json:"id"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Color        string     `json:"color,omitempty"`
	LecturerID   *uuid.UUID `json:"lecturer_id,omitempty"`
	OwnerUserID  *uuid.UUID `json:"owner_user_id,omitempty"`
	LecturerName string     `json:"lecturer_name,omitempty"`
}

type QuestionView struct {
	ID             uuid.UUID            `json:"id"`
	TenantID       uuid.UUID            `json:"tenant_id"`
	Code           string               `json:"code"`
	Name           string               `json:"name"`
	QuestionBankID uuid.UUID            `json:"question_bank_id"`
	LecturerID     *uuid.UUID           `json:"lecturer_id,omitempty"`
	OwnerUserID    *uuid.UUID           `json:"owner_user_id,omitempty"`
	QuestionText   string               `json:"question_text"`
	QuestionType   string               `json:"question_type"`
	AnswerMode     string               `json:"answer_mode"`
	Difficulty     string               `json:"difficulty"`
	Score          float64              `json:"score"`
	Explanation    string               `json:"explanation,omitempty"`
	Status         string               `json:"status"`
	Options        []QuestionOptionView `json:"options"`
	Media          []QuestionMediaView  `json:"media,omitempty"`
	Tags           []QuestionTagView    `json:"tags"`
	TagIDs         []uuid.UUID          `json:"tag_ids"`
	Metadata       map[string]any       `json:"metadata,omitempty"`
}

type QuestionUsageView struct {
	QuestionID           uuid.UUID         `json:"question_id"`
	Version              int               `json:"version"`
	ExamSessionCount     int64             `json:"exam_session_count"`
	ExamCount            int64             `json:"exam_count"`
	AnswerCount          int64             `json:"answer_count"`
	GradingCount         int64             `json:"grading_count"`
	CanHardDelete        bool              `json:"can_hard_delete"`
	RecommendedAction    string            `json:"recommended_action"`
	UsedInExamSessionIDs []uuid.UUID       `json:"used_in_exam_session_ids,omitempty"`
	UsedInExamSummaries  []QuestionExamRef `json:"used_in_exam_summaries,omitempty"`
}

type QuestionVersionView struct {
	ID          uuid.UUID      `json:"id"`
	QuestionID  uuid.UUID      `json:"question_id"`
	Version     int            `json:"version"`
	ActorUserID *uuid.UUID     `json:"actor_user_id,omitempty"`
	EventType   string         `json:"event_type"`
	Snapshot    map[string]any `json:"snapshot"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   string         `json:"created_at"`
}

type QuestionVersionListResult struct {
	Items []QuestionVersionView `json:"items"`
	Total int                   `json:"total"`
}

type QuestionExamRef struct {
	ExamID        uuid.UUID `json:"exam_id"`
	ExamName      string    `json:"exam_name"`
	SessionCount  int64     `json:"session_count"`
	LastStartedAt string    `json:"last_started_at,omitempty"`
}
