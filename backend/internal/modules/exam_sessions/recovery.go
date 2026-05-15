package exam_sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
)

type recoveryPolicy struct {
	TimerMode                      string `json:"timer_mode"`
	MaxPauseSeconds                int64  `json:"max_pause_seconds"`
	MaxReconnectAttempts           int    `json:"max_reconnect_attempts"`
	DeviceChangeRequiresApproval   bool   `json:"device_change_requires_approval"`
	AutoSubmitWhenRecoveryExceeded bool   `json:"auto_submit_when_recovery_exceeded"`
}

type antiCheatPolicy struct {
	FullscreenRequired      bool    `json:"fullscreen_required"`
	WebcamRequired          bool    `json:"webcam_required"`
	BlockCopyPaste          bool    `json:"block_copy_paste"`
	BlockRightClick         bool    `json:"block_right_click"`
	SnapshotIntervalSeconds int64   `json:"snapshot_interval_seconds"`
	CriticalScoreThreshold  float64 `json:"critical_score_threshold"`
}

func defaultRecoveryPolicy() recoveryPolicy {
	return recoveryPolicy{
		TimerMode:                      "recovery_pause",
		MaxPauseSeconds:                3600,
		MaxReconnectAttempts:           3,
		DeviceChangeRequiresApproval:   true,
		AutoSubmitWhenRecoveryExceeded: false,
	}
}

func defaultAntiCheatPolicy() antiCheatPolicy {
	return antiCheatPolicy{
		FullscreenRequired:      true,
		WebcamRequired:          true,
		BlockCopyPaste:          true,
		BlockRightClick:         true,
		SnapshotIntervalSeconds: 60,
		CriticalScoreThreshold:  90,
	}
}

func recoveryPolicyFromMetadata(metadata map[string]any) recoveryPolicy {
	policy := defaultRecoveryPolicy()
	if raw, ok := metadata["recovery_policy"]; ok {
		applyJSON(raw, &policy)
	}
	if policy.TimerMode == "" {
		policy.TimerMode = "recovery_pause"
	}
	if policy.MaxPauseSeconds <= 0 {
		policy.MaxPauseSeconds = 3600
	}
	if policy.MaxReconnectAttempts <= 0 {
		policy.MaxReconnectAttempts = 3
	}
	return policy
}

func antiCheatPolicyFromMetadata(metadata map[string]any) antiCheatPolicy {
	policy := defaultAntiCheatPolicy()
	if raw, ok := metadata["anti_cheat_policy"]; ok {
		applyJSON(raw, &policy)
	}
	if policy.SnapshotIntervalSeconds <= 0 {
		policy.SnapshotIntervalSeconds = 60
	}
	if policy.CriticalScoreThreshold <= 0 {
		policy.CriticalScoreThreshold = 90
	}
	return policy
}

func initialSessionMetadata(recoveryPolicy recoveryPolicy, antiCheatPolicy antiCheatPolicy, req StartExamRequest, durationMinutes int) map[string]any {
	totalSeconds := int64(durationMinutes * 60)
	return map[string]any{
		"recovery_policy":   recoveryPolicy,
		"anti_cheat_policy": antiCheatPolicy,
		"device": map[string]any{
			"fingerprint": req.DeviceFingerprint,
			"name":        req.DeviceName,
			"user_agent":  req.UserAgent,
		},
		"recovery": map[string]any{
			"timer_mode":          recoveryPolicy.TimerMode,
			"timer_paused":        false,
			"remaining_seconds":   totalSeconds,
			"total_pause_seconds": int64(0),
			"reconnect_count":     0,
			"status":              "active",
			"review_required":     false,
		},
		"anti_cheat": map[string]any{
			"suspicious_score": float64(0),
			"status":           "normal",
		},
	}
}

func applyRecoveryState(state *ExamSessionState, metadata map[string]any) {
	recovery := mapFromAny(metadata["recovery"])
	policy := recoveryPolicyFromMetadata(metadata)
	state.TimerMode = stringFromAny(recovery["timer_mode"], policy.TimerMode)
	state.TimerPaused = boolFromAny(recovery["timer_paused"])
	state.RecoveryStatus = stringFromAny(recovery["status"], "")
	state.ReviewRequired = boolFromAny(recovery["review_required"])
	state.ReconnectCount = int(numberFromAny(recovery["reconnect_count"]))
	state.TotalPauseSeconds = int64(numberFromAny(recovery["total_pause_seconds"]))
	if state.TimerPaused {
		state.RemainingSeconds = int64(math.Max(0, numberFromAny(recovery["remaining_seconds"])))
	}
	antiCheat := mapFromAny(metadata["anti_cheat"])
	state.SuspiciousScore = numberFromAny(antiCheat["suspicious_score"])
}

func (s *service) writeTimerRedis(ctx context.Context, state ExamSessionState) error {
	if s.deps.Redis == nil {
		return nil
	}
	return s.deps.Redis.HSet(ctx, "exam_session:"+state.SessionID.String(),
		"status", state.Status,
		"ends_at", state.EndsAt.Format(time.RFC3339Nano),
		"remaining_seconds", state.RemainingSeconds,
		"timer_mode", state.TimerMode,
		"timer_paused", strconv.FormatBool(state.TimerPaused),
		"recovery_status", state.RecoveryStatus,
		"review_required", strconv.FormatBool(state.ReviewRequired),
		"reconnect_count", state.ReconnectCount,
		"total_pause_seconds", state.TotalPauseSeconds,
		"suspicious_score", state.SuspiciousScore,
	).Err()
}

func MarkSessionDisconnected(ctx context.Context, deps shared.Deps, sessionID uuid.UUID) {
	if deps.DB == nil || sessionID == uuid.Nil {
		return
	}
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	var tenantID uuid.UUID
	var status string
	var endsAt time.Time
	var rawMetadata []byte
	err = tx.QueryRow(ctx, `
		SELECT tenant_id, status_enum::text, ends_at, COALESCE(metadata,'{}'::jsonb)
		FROM exam_sessions
		WHERE id=$1 AND deleted_at IS NULL
		FOR UPDATE`, sessionID).Scan(&tenantID, &status, &endsAt, &rawMetadata)
	if err != nil || status != "started" {
		return
	}
	now := time.Now().UTC()
	metadata := map[string]any{}
	_ = json.Unmarshal(rawMetadata, &metadata)
	policy := recoveryPolicyFromMetadata(metadata)
	recovery := mapFromAny(metadata["recovery"])
	remaining := int64(math.Max(0, endsAt.Sub(now).Seconds()))
	if boolFromAny(recovery["timer_paused"]) {
		remaining = int64(math.Max(0, numberFromAny(recovery["remaining_seconds"])))
	}
	recovery["timer_mode"] = policy.TimerMode
	recovery["disconnected_at"] = now.Format(time.RFC3339Nano)
	recovery["last_disconnect_at"] = now.Format(time.RFC3339Nano)
	recovery["disconnect_count"] = int(numberFromAny(recovery["disconnect_count"])) + 1
	if policy.TimerMode == "recovery_pause" {
		recovery["timer_paused"] = true
		recovery["pause_started_at"] = now.Format(time.RFC3339Nano)
		recovery["remaining_seconds"] = remaining
		recovery["status"] = "paused"
	} else {
		recovery["timer_paused"] = false
		recovery["remaining_seconds"] = remaining
		recovery["status"] = "reconnecting"
	}
	metadata["recovery"] = recovery
	raw, _ := json.Marshal(metadata)
	_, _ = tx.Exec(ctx, `
		UPDATE exam_sessions
		SET status='reconnecting', status_enum='reconnecting', metadata=$1, updated_at=now()
		WHERE id=$2 AND tenant_id=$3`, raw, sessionID, tenantID)
	_, _ = tx.Exec(ctx, `
		INSERT INTO recovery_logs(id,tenant_id,code,name,status,exam_session_id,event_type,client_time,metadata)
		VALUES($1,$2,$3,'Timer paused after disconnect','active',$4,'disconnect_detected',$5,$6)`,
		uuid.New(), tenantID, "RCV-"+sessionID.String()[:8], sessionID, now, raw)
	_ = tx.Commit(ctx)

	if deps.Redis != nil {
		_ = deps.Redis.HSet(ctx, "exam_session:"+sessionID.String(),
			"status", "reconnecting",
			"timer_mode", policy.TimerMode,
			"timer_paused", strconv.FormatBool(policy.TimerMode == "recovery_pause"),
			"remaining_seconds", remaining,
			"recovery_status", recovery["status"],
		).Err()
	}
}

func (s *service) resumeRecoveryTimer(ctx context.Context, tenantID uuid.UUID, req ReconnectRequest) error {
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	var endsAt time.Time
	var rawMetadata []byte
	err = tx.QueryRow(ctx, `
		SELECT status_enum::text, ends_at, COALESCE(metadata,'{}'::jsonb)
		FROM exam_sessions
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL
		FOR UPDATE`, req.SessionID, tenantID).Scan(&status, &endsAt, &rawMetadata)
	if err != nil {
		return err
	}
	if status == "completed" {
		return tx.Commit(ctx)
	}
	metadata := map[string]any{}
	_ = json.Unmarshal(rawMetadata, &metadata)
	policy := recoveryPolicyFromMetadata(metadata)
	recovery := mapFromAny(metadata["recovery"])
	device := mapFromAny(metadata["device"])
	antiCheat := mapFromAny(metadata["anti_cheat"])
	now := time.Now().UTC()
	remaining := int64(math.Max(0, endsAt.Sub(now).Seconds()))
	if boolFromAny(recovery["timer_paused"]) {
		remaining = int64(math.Max(0, numberFromAny(recovery["remaining_seconds"])))
	}
	pauseStartedAt := timeFromAny(recovery["pause_started_at"])
	pauseSeconds := int64(0)
	if !pauseStartedAt.IsZero() && boolFromAny(recovery["timer_paused"]) {
		pauseSeconds = int64(math.Max(0, now.Sub(pauseStartedAt).Seconds()))
	}
	totalPauseSeconds := int64(numberFromAny(recovery["total_pause_seconds"])) + pauseSeconds
	reconnectCount := int(numberFromAny(recovery["reconnect_count"])) + 1
	storedFingerprint := stringFromAny(device["fingerprint"], "")
	deviceChanged := storedFingerprint != "" && req.DeviceFingerprint != "" && storedFingerprint != req.DeviceFingerprint
	reviewRequired := boolFromAny(recovery["review_required"])
	if policy.MaxPauseSeconds > 0 && totalPauseSeconds > policy.MaxPauseSeconds {
		reviewRequired = true
	}
	if policy.MaxReconnectAttempts > 0 && reconnectCount > policy.MaxReconnectAttempts {
		reviewRequired = true
	}
	if deviceChanged && policy.DeviceChangeRequiresApproval {
		reviewRequired = true
	}
	if req.DeviceFingerprint != "" {
		device["last_fingerprint"] = req.DeviceFingerprint
	}
	if req.DeviceName != "" {
		device["last_name"] = req.DeviceName
	}
	if req.UserAgent != "" {
		device["last_user_agent"] = req.UserAgent
	}
	device["changed"] = deviceChanged
	recovery["timer_paused"] = false
	recovery["remaining_seconds"] = remaining
	recovery["resumed_at"] = now.Format(time.RFC3339Nano)
	recovery["last_pause_seconds"] = pauseSeconds
	recovery["total_pause_seconds"] = totalPauseSeconds
	recovery["reconnect_count"] = reconnectCount
	recovery["review_required"] = reviewRequired
	recovery["status"] = "resumed"
	if reviewRequired {
		recovery["status"] = "requires_review"
	}
	score := numberFromAny(antiCheat["suspicious_score"])
	if pauseSeconds > 120 || reconnectCount > 1 {
		score += 20
	}
	if deviceChanged {
		score += 40
	}
	antiCheat["suspicious_score"] = score
	if score >= 90 {
		antiCheat["status"] = "critical"
	} else if score >= 60 {
		antiCheat["status"] = "suspicious"
	} else if score >= 30 {
		antiCheat["status"] = "warning"
	} else {
		antiCheat["status"] = "normal"
	}
	metadata["device"] = device
	metadata["recovery"] = recovery
	metadata["anti_cheat"] = antiCheat
	raw, _ := json.Marshal(metadata)
	newEndsAt := now.Add(time.Duration(remaining) * time.Second)
	statusText := "started"
	if reviewRequired {
		statusText = "requires_review"
	}
	_, err = tx.Exec(ctx, `
		UPDATE exam_sessions
		SET status=$1, status_enum='started', ends_at=$2, metadata=$3, updated_at=now()
		WHERE id=$4 AND tenant_id=$5`, statusText, newEndsAt, raw, req.SessionID, tenantID)
	if err != nil {
		return err
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO reconnect_logs(id,tenant_id,code,name,status,exam_session_id,disconnected_at,reconnected_at,auto_submitted,metadata)
		VALUES($1,$2,$3,'Reconnect','active',$4,$5,$6,false,$7)`,
		uuid.New(), tenantID, "REC-"+req.SessionID.String()[:8], req.SessionID, timeFromAny(recovery["last_disconnect_at"]), now, raw)
	_, _ = tx.Exec(ctx, `
		INSERT INTO recovery_logs(id,tenant_id,code,name,status,exam_session_id,event_type,client_time,metadata)
		VALUES($1,$2,$3,'Timer resumed after reconnect','active',$4,'reconnect_resume',$5,$6)`,
		uuid.New(), tenantID, "RCV-"+req.SessionID.String()[:8], req.SessionID, now, raw)
	if pauseSeconds > 120 || reconnectCount > 1 {
		_, _ = tx.Exec(ctx, `
			INSERT INTO proctoring_logs(id,tenant_id,code,name,status,exam_session_id,event_type,score,metadata)
			VALUES($1,$2,$3,'Abnormal reconnect','active',$4,'abnormal_reconnect',20,$5)`,
			uuid.New(), tenantID, "PRC-"+req.SessionID.String()[:8], req.SessionID, raw)
	}
	if deviceChanged {
		_, _ = tx.Exec(ctx, `
			INSERT INTO proctoring_logs(id,tenant_id,code,name,status,exam_session_id,event_type,score,metadata)
			VALUES($1,$2,$3,'Device changed','active',$4,'device_changed',40,$5)`,
			uuid.New(), tenantID, "PRC-"+req.SessionID.String()[:8], req.SessionID, raw)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if s.deps.Redis != nil {
		_ = s.deps.Redis.HSet(ctx, "exam_session:"+req.SessionID.String(),
			"status", statusText,
			"ends_at", newEndsAt.Format(time.RFC3339Nano),
			"timer_paused", "false",
			"remaining_seconds", remaining,
			"recovery_status", recovery["status"],
			"review_required", strconv.FormatBool(reviewRequired),
			"reconnect_count", reconnectCount,
			"total_pause_seconds", totalPauseSeconds,
			"suspicious_score", score,
		).Err()
	}
	return nil
}

func applyJSON(input any, output any) {
	raw, err := json.Marshal(input)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, output)
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	out := map[string]any{}
	applyJSON(value, &out)
	return out
}

func numberFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		out, _ := typed.Float64()
		return out
	case string:
		out, _ := strconv.ParseFloat(typed, 64)
		return out
	default:
		return 0
	}
}

func boolFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		out, _ := strconv.ParseBool(typed)
		return out
	default:
		return false
	}
}

func stringFromAny(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	out := strings.TrimSpace(fmt.Sprint(value))
	if out == "" || out == "<nil>" {
		return fallback
	}
	return out
}

func timeFromAny(value any) time.Time {
	raw := stringFromAny(value, "")
	if raw == "" {
		return time.Time{}
	}
	out, _ := time.Parse(time.RFC3339Nano, raw)
	return out
}
