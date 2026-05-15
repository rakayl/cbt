package dashboard

import (
	"time"

	"github.com/google/uuid"
)

type MetricView struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Value   float64 `json:"value"`
	Display string  `json:"display"`
	Tone    string  `json:"tone"`
}

type DashboardActivityView struct {
	ID        uuid.UUID      `json:"id"`
	Title     string         `json:"title"`
	Subtitle  string         `json:"subtitle,omitempty"`
	Type      string         `json:"type"`
	Severity  string         `json:"severity,omitempty"`
	Score     float64        `json:"score,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type DashboardExamView struct {
	ID             uuid.UUID      `json:"id"`
	Code           string         `json:"code"`
	Name           string         `json:"name"`
	Status         string         `json:"status"`
	StartsAt       *time.Time     `json:"starts_at,omitempty"`
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
	InvitationCode string         `json:"invitation_code,omitempty"`
	ParticipantCnt int64          `json:"participant_count,omitempty"`
	CompletedCnt   int64          `json:"completed_count,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type DashboardResultView struct {
	SessionID   uuid.UUID  `json:"session_id"`
	ExamName    string     `json:"exam_name"`
	Code        string     `json:"code"`
	Status      string     `json:"status"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	Score       float64    `json:"score"`
	MaxScore    float64    `json:"max_score"`
	Percentage  float64    `json:"percentage"`
	Passed      bool       `json:"passed"`
}

type DashboardTrendPoint struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

type DashboardSummaryResponse struct {
	Role          string                  `json:"role"`
	TenantID      uuid.UUID               `json:"tenant_id,omitempty"`
	GeneratedAt   time.Time               `json:"generated_at"`
	Metrics       []MetricView            `json:"metrics"`
	UpcomingExams []DashboardExamView     `json:"upcoming_exams,omitempty"`
	ActiveExams   []DashboardExamView     `json:"active_exams,omitempty"`
	RecentResults []DashboardResultView   `json:"recent_results,omitempty"`
	Activities    []DashboardActivityView `json:"activities,omitempty"`
	Trends        []DashboardTrendPoint   `json:"trends,omitempty"`
	Health        map[string]string       `json:"health,omitempty"`
}
