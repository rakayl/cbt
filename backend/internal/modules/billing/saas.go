package billing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type UsageSummary struct {
	TenantID              uuid.UUID      `json:"tenant_id"`
	PlanCode              string         `json:"plan_code"`
	PlanName              string         `json:"plan_name"`
	MaxStudents           int            `json:"max_students"`
	MaxConcurrentExams    int            `json:"max_concurrent_exams"`
	Students              int            `json:"students"`
	ActiveExamSessions    int            `json:"active_exam_sessions"`
	StorageObjects        int            `json:"storage_objects"`
	Features              map[string]any `json:"features"`
	StudentQuotaPercent   float64        `json:"student_quota_percent"`
	ExamQuotaPercent      float64        `json:"exam_quota_percent"`
	PaymentIntegrationURL string         `json:"payment_integration_url,omitempty"`
}

type CheckoutIntentRequest struct {
	PlanCode     string `json:"plan_code" validate:"required"`
	BillingEmail string `json:"billing_email" validate:"required,email"`
}

type CheckoutIntent struct {
	IntentID    uuid.UUID `json:"intent_id"`
	PlanCode    string    `json:"plan_code"`
	Status      string    `json:"status"`
	RedirectURL string    `json:"redirect_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type ChangePlanRequest struct {
	PlanCode string `json:"plan_code" validate:"required"`
}

func (s *service) Usage(ctx context.Context, tenantID uuid.UUID) (UsageSummary, error) {
	var out UsageSummary
	var featuresRaw []byte
	err := s.deps.DB.QueryRow(ctx, `
		SELECT t.id, COALESCE(sp.code,'FREE'), COALESCE(sp.name,'Free'), COALESCE(sp.max_students,100),
		       COALESCE(sp.max_concurrent_exams,10), COALESCE(sp.features,'{}'::jsonb)
		FROM tenants t
		LEFT JOIN subscription_plans sp ON sp.id=t.subscription_plan_id AND sp.deleted_at IS NULL
		WHERE t.id=$1 AND t.deleted_at IS NULL`, tenantID).
		Scan(&out.TenantID, &out.PlanCode, &out.PlanName, &out.MaxStudents, &out.MaxConcurrentExams, &featuresRaw)
	if err != nil {
		return UsageSummary{}, err
	}
	_ = json.Unmarshal(featuresRaw, &out.Features)
	_ = s.deps.DB.QueryRow(ctx, `SELECT count(*) FROM students WHERE tenant_id=$1 AND deleted_at IS NULL`, tenantID).Scan(&out.Students)
	_ = s.deps.DB.QueryRow(ctx, `SELECT count(*) FROM exam_sessions WHERE tenant_id=$1 AND deleted_at IS NULL AND status_enum IN ('started','reconnecting')`, tenantID).Scan(&out.ActiveExamSessions)
	_ = s.deps.DB.QueryRow(ctx, `SELECT count(*) FROM screen_recordings WHERE tenant_id=$1 AND deleted_at IS NULL`, tenantID).Scan(&out.StorageObjects)
	out.StudentQuotaPercent = quotaPercent(out.Students, out.MaxStudents)
	out.ExamQuotaPercent = quotaPercent(out.ActiveExamSessions, out.MaxConcurrentExams)
	return out, nil
}

func (s *service) CreateCheckoutIntent(ctx context.Context, tenantID uuid.UUID, req CheckoutIntentRequest) (CheckoutIntent, error) {
	if err := validate.Struct(req); err != nil {
		return CheckoutIntent{}, err
	}
	var planID uuid.UUID
	if err := s.deps.DB.QueryRow(ctx, `SELECT id FROM subscription_plans WHERE code=$1 AND deleted_at IS NULL`, req.PlanCode).Scan(&planID); err != nil {
		return CheckoutIntent{}, err
	}
	intent := CheckoutIntent{IntentID: uuid.New(), PlanCode: req.PlanCode, Status: "requires_payment_provider", RedirectURL: "/billing/checkout/" + req.PlanCode, ExpiresAt: time.Now().UTC().Add(30 * time.Minute)}
	raw, _ := json.Marshal(map[string]any{"billing_email": req.BillingEmail, "plan_id": planID, "redirect_url": intent.RedirectURL})
	_, err := s.deps.DB.Exec(ctx, `
		INSERT INTO billing(id,tenant_id,code,name,status,metadata)
		VALUES($1,$2,$3,'Checkout intent',$4,$5)`,
		intent.IntentID, tenantID, "CHK-"+intent.IntentID.String()[:8], intent.Status, raw)
	if err != nil {
		return CheckoutIntent{}, err
	}
	if s.deps.Rabbit != nil {
		body, _ := json.Marshal(intent)
		_ = s.deps.Rabbit.Publish(ctx, "billing.checkout_intent", body)
	}
	return intent, nil
}

func (s *service) ChangePlan(ctx context.Context, tenantID uuid.UUID, req ChangePlanRequest) (UsageSummary, error) {
	if err := validate.Struct(req); err != nil {
		return UsageSummary{}, err
	}
	var planID uuid.UUID
	if err := s.deps.DB.QueryRow(ctx, `SELECT id FROM subscription_plans WHERE code=$1 AND deleted_at IS NULL`, req.PlanCode).Scan(&planID); err != nil {
		return UsageSummary{}, err
	}
	tag, err := s.deps.DB.Exec(ctx, `UPDATE tenants SET subscription_plan_id=$1, updated_at=now() WHERE id=$2 AND deleted_at IS NULL`, planID, tenantID)
	if err != nil {
		return UsageSummary{}, err
	}
	if tag.RowsAffected() == 0 {
		return UsageSummary{}, errors.New("tenant not found")
	}
	return s.Usage(ctx, tenantID)
}

func quotaPercent(used, limit int) float64 {
	if limit <= 0 {
		return 100
	}
	return float64(used) / float64(limit) * 100
}
