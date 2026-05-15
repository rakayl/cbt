package proctoring

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type ProctoringEventRequest struct {
	ExamSessionID uuid.UUID      `json:"exam_session_id" validate:"required"`
	EventType     string         `json:"event_type" validate:"required"`
	Severity      string         `json:"severity" validate:"omitempty,oneof=low medium high critical"`
	Metadata      map[string]any `json:"metadata"`
	ClientTime    time.Time      `json:"client_time"`
}

type ProctoringEventResult struct {
	Logged          bool    `json:"logged"`
	EventType       string  `json:"event_type"`
	SuspiciousScore float64 `json:"suspicious_score"`
}

type ProctoringSnapshotRequest struct {
	ExamSessionID uuid.UUID      `json:"exam_session_id" validate:"required"`
	EventType     string         `json:"event_type" validate:"omitempty"`
	Severity      string         `json:"severity" validate:"omitempty,oneof=low medium high critical"`
	ImageData     string         `json:"image_data" validate:"required"`
	FaceCount     *int           `json:"face_count"`
	Metadata      map[string]any `json:"metadata"`
	ClientTime    time.Time      `json:"client_time"`
}

type ProctoringSnapshotResult struct {
	Logged          bool    `json:"logged"`
	ObjectKey       string  `json:"object_key"`
	EventType       string  `json:"event_type"`
	SuspiciousScore float64 `json:"suspicious_score"`
}

type ProctoringEventView struct {
	ID            uuid.UUID      `json:"id"`
	ExamSessionID *uuid.UUID     `json:"exam_session_id,omitempty"`
	EventType     string         `json:"event_type"`
	Severity      string         `json:"severity"`
	Score         float64        `json:"score"`
	Status        string         `json:"status"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
}

type ProctoringTimelineResponse struct {
	ExamSessionID   uuid.UUID                `json:"exam_session_id"`
	StudentID       *uuid.UUID               `json:"student_id,omitempty"`
	StudentName     string                   `json:"student_name,omitempty"`
	ExamID          *uuid.UUID               `json:"exam_id,omitempty"`
	ExamName        string                   `json:"exam_name,omitempty"`
	Status          string                   `json:"status,omitempty"`
	SuspiciousScore float64                  `json:"suspicious_score"`
	Summary         map[string]int           `json:"summary"`
	Items           []ProctoringTimelineItem `json:"items"`
}

type ProctoringTimelineItem struct {
	ID        uuid.UUID      `json:"id"`
	Source    string         `json:"source"`
	EventType string         `json:"event_type"`
	Severity  string         `json:"severity"`
	Score     float64        `json:"score"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

func (s *service) IngestEvent(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, req ProctoringEventRequest) (ProctoringEventResult, error) {
	if err := validate.Struct(req); err != nil {
		return ProctoringEventResult{}, err
	}
	if err := s.ensureExamSessionAccess(ctx, tenantID, actorUserID, permissions, req.ExamSessionID); err != nil {
		return ProctoringEventResult{}, err
	}
	score := suspiciousScore(req.EventType, req.Severity)
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	req.Metadata["severity"] = req.Severity
	req.Metadata["client_time"] = req.ClientTime
	raw, _ := json.Marshal(req.Metadata)
	code := "PRC-" + uuid.NewString()[:8]

	_, err := s.deps.DB.Exec(ctx, `
		INSERT INTO proctoring_logs(id,tenant_id,code,name,status,exam_session_id,event_type,score,metadata)
		VALUES($1,$2,$3,$4,'active',$5,$6,$7,$8)`,
		uuid.New(), tenantID, code, req.EventType, req.ExamSessionID, req.EventType, score, raw)
	if err != nil {
		return ProctoringEventResult{}, err
	}
	totalScore := s.addSuspiciousScore(ctx, tenantID, req.ExamSessionID, score)

	switch req.EventType {
	case "tab_switch", "fullscreen_exit", "copy_paste", "right_click", "devtools_suspected", "multiple_monitor":
		_, _ = s.deps.DB.Exec(ctx, `
			INSERT INTO browser_activity_logs(id,tenant_id,code,name,status,exam_session_id,activity_type,metadata)
			VALUES($1,$2,$3,$4,'active',$5,$6,$7)`,
			uuid.New(), tenantID, "BRW-"+uuid.NewString()[:8], req.EventType, req.ExamSessionID, req.EventType, raw)
	case "no_face", "multiple_face", "face_detected", "head_pose", "audio_noise", "webcam_snapshot", "webcam_unavailable":
		faceCount := 0
		if v, ok := req.Metadata["face_count"].(float64); ok {
			faceCount = int(v)
		}
		_, _ = s.deps.DB.Exec(ctx, `
			INSERT INTO face_detection_logs(id,tenant_id,code,name,status,exam_session_id,face_count,metadata)
			VALUES($1,$2,$3,$4,'active',$5,$6,$7)`,
			uuid.New(), tenantID, "FAC-"+uuid.NewString()[:8], req.EventType, req.ExamSessionID, faceCount, raw)
	}

	if s.deps.Redis != nil {
		_ = s.deps.Redis.HSet(ctx, "proctoring:"+req.ExamSessionID.String(), req.EventType, fmt.Sprintf("%.2f", score), "suspicious_score", fmt.Sprintf("%.2f", totalScore)).Err()
	}
	if s.deps.Rabbit != nil {
		body, _ := json.Marshal(map[string]any{"tenant_id": tenantID, "exam_session_id": req.ExamSessionID, "event_type": req.EventType, "score": score})
		_ = s.deps.Rabbit.Publish(ctx, "proctoring_queue", body)
		_ = s.deps.Rabbit.Publish(ctx, "analytics_queue", body)
	}
	return ProctoringEventResult{Logged: true, EventType: req.EventType, SuspiciousScore: totalScore}, nil
}

func (s *service) UploadSnapshot(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, req ProctoringSnapshotRequest) (ProctoringSnapshotResult, error) {
	if err := validate.Struct(req); err != nil {
		return ProctoringSnapshotResult{}, err
	}
	if err := s.ensureExamSessionAccess(ctx, tenantID, actorUserID, permissions, req.ExamSessionID); err != nil {
		return ProctoringSnapshotResult{}, err
	}
	if s.deps.Storage == nil {
		return ProctoringSnapshotResult{}, fmt.Errorf("object storage is not configured")
	}
	eventType := strings.TrimSpace(req.EventType)
	if eventType == "" {
		eventType = "webcam_snapshot"
	}
	severity := strings.TrimSpace(req.Severity)
	if severity == "" {
		severity = "low"
	}
	imageBytes, contentType, err := decodeSnapshot(req.ImageData)
	if err != nil {
		return ProctoringSnapshotResult{}, err
	}
	extension := ".jpg"
	if contentType == "image/png" {
		extension = ".png"
	}
	objectKey := filepath.ToSlash(fmt.Sprintf("tenants/%s/exam-sessions/%s/proctoring/%s-%s%s", tenantID, req.ExamSessionID, time.Now().UTC().Format("20060102T150405.000000000"), uuid.NewString()[:8], extension))
	_, err = s.deps.Storage.PutObject(ctx, s.deps.Config.S3Bucket, objectKey, bytes.NewReader(imageBytes), int64(len(imageBytes)), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return ProctoringSnapshotResult{}, err
	}
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["severity"] = severity
	metadata["client_time"] = req.ClientTime
	metadata["object_key"] = objectKey
	metadata["content_type"] = contentType
	metadata["size_bytes"] = len(imageBytes)
	if req.FaceCount != nil {
		metadata["face_count"] = *req.FaceCount
	}
	raw, _ := json.Marshal(metadata)
	score := suspiciousScore(eventType, severity)
	_, err = s.deps.DB.Exec(ctx, `
		INSERT INTO proctoring_logs(id,tenant_id,code,name,status,exam_session_id,event_type,score,metadata)
		VALUES($1,$2,$3,$4,'active',$5,$6,$7,$8)`,
		uuid.New(), tenantID, "PRC-"+uuid.NewString()[:8], eventType, req.ExamSessionID, eventType, score, raw)
	if err != nil {
		return ProctoringSnapshotResult{}, err
	}
	totalScore := s.addSuspiciousScore(ctx, tenantID, req.ExamSessionID, score)
	faceCount := 0
	if req.FaceCount != nil {
		faceCount = *req.FaceCount
	}
	_, _ = s.deps.DB.Exec(ctx, `
		INSERT INTO face_detection_logs(id,tenant_id,code,name,status,exam_session_id,face_count,metadata)
		VALUES($1,$2,$3,$4,'active',$5,$6,$7)`,
		uuid.New(), tenantID, "FAC-"+uuid.NewString()[:8], eventType, req.ExamSessionID, faceCount, raw)
	if s.deps.Redis != nil {
		_ = s.deps.Redis.HSet(ctx, "proctoring:"+req.ExamSessionID.String(), eventType, fmt.Sprintf("%.2f", score), "last_snapshot", objectKey, "suspicious_score", fmt.Sprintf("%.2f", totalScore)).Err()
	}
	if s.deps.Rabbit != nil {
		body, _ := json.Marshal(map[string]any{"tenant_id": tenantID, "exam_session_id": req.ExamSessionID, "event_type": eventType, "score": score, "object_key": objectKey})
		_ = s.deps.Rabbit.Publish(ctx, "proctoring_queue", body)
		_ = s.deps.Rabbit.Publish(ctx, "analytics_queue", body)
	}
	return ProctoringSnapshotResult{Logged: true, ObjectKey: objectKey, EventType: eventType, SuspiciousScore: totalScore}, nil
}

func (s *service) ListEvents(ctx context.Context, tenantID uuid.UUID, q pagination.Query, eventType, severity, sessionID string) (shared.ListResult, error) {
	where := "deleted_at IS NULL"
	args := []any{}
	i := 1
	if tenantID != uuid.Nil {
		where += fmt.Sprintf(" AND tenant_id=$%d", i)
		args = append(args, tenantID)
		i++
	}
	if eventType != "" {
		where += fmt.Sprintf(" AND event_type=$%d", i)
		args = append(args, eventType)
		i++
	}
	if severity != "" {
		where += fmt.Sprintf(" AND metadata->>'severity'=$%d", i)
		args = append(args, severity)
		i++
	}
	if sessionID != "" {
		if parsed, err := uuid.Parse(sessionID); err == nil {
			where += fmt.Sprintf(" AND exam_session_id=$%d", i)
			args = append(args, parsed)
			i++
		}
	}
	if q.Search != "" {
		where += fmt.Sprintf(" AND lower(event_type) LIKE $%d", i)
		args = append(args, "%"+strings.ToLower(q.Search)+"%")
		i++
	}
	var total int64
	if err := s.deps.DB.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM proctoring_logs WHERE %s", where), args...).Scan(&total); err != nil {
		return shared.ListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	rows, err := s.deps.DB.Query(ctx, fmt.Sprintf(`
		SELECT id, coalesce(exam_session_id::text,''), event_type, score, status, metadata, created_at
		FROM proctoring_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, i, i+1), args...)
	if err != nil {
		return shared.ListResult{}, err
	}
	defer rows.Close()
	items := []shared.Record{}
	for rows.Next() {
		var id uuid.UUID
		var examSessionID string
		var event string
		var score float64
		var status string
		var raw []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &examSessionID, &event, &score, &status, &raw, &createdAt); err != nil {
			return shared.ListResult{}, err
		}
		metadata := map[string]any{}
		_ = json.Unmarshal(raw, &metadata)
		severity, _ := metadata["severity"].(string)
		if severity == "" {
			severity = severityFromScore(score)
		}
		items = append(items, shared.Record{
			BaseModel: shared.BaseModel{ID: id, TenantID: tenantID, CreatedAt: createdAt, UpdatedAt: createdAt},
			Code:      examSessionID,
			Name:      event,
			Status:    status,
			Metadata: map[string]any{
				"exam_session_id": examSessionID,
				"event_type":      event,
				"severity":        severity,
				"score":           score,
				"detail":          metadata,
			},
		})
	}
	return shared.ListResult{Items: items, Page: q.Page, Limit: q.Limit, Total: total}, rows.Err()
}

func (s *service) SessionTimeline(ctx context.Context, tenantID, sessionID uuid.UUID) (ProctoringTimelineResponse, error) {
	out := ProctoringTimelineResponse{ExamSessionID: sessionID, Summary: map[string]int{}}
	var studentID, examID uuid.UUID
	var rawMetadata []byte
	err := s.deps.DB.QueryRow(ctx, `
		SELECT es.student_id, COALESCE(st.name,''), es.exam_id, COALESCE(e.name,''), es.status_enum::text, COALESCE(es.metadata,'{}'::jsonb)
		FROM exam_sessions es
		LEFT JOIN students st ON st.id=es.student_id AND st.deleted_at IS NULL
		LEFT JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
		WHERE es.id=$1 AND es.tenant_id=$2 AND es.deleted_at IS NULL`,
		sessionID, tenantID).Scan(&studentID, &out.StudentName, &examID, &out.ExamName, &out.Status, &rawMetadata)
	if err != nil {
		return out, err
	}
	out.StudentID = &studentID
	out.ExamID = &examID
	sessionMetadata := map[string]any{}
	_ = json.Unmarshal(rawMetadata, &sessionMetadata)
	out.SuspiciousScore = numberFromAny(mapFromAny(sessionMetadata["anti_cheat"])["suspicious_score"])

	rows, err := s.deps.DB.Query(ctx, `
		SELECT id, 'proctoring' AS source, event_type, score, COALESCE(metadata,'{}'::jsonb), created_at
		FROM proctoring_logs
		WHERE tenant_id=$1 AND exam_session_id=$2 AND deleted_at IS NULL
		UNION ALL
		SELECT id, 'browser' AS source, activity_type AS event_type, 0 AS score, COALESCE(metadata,'{}'::jsonb), created_at
		FROM browser_activity_logs
		WHERE tenant_id=$1 AND exam_session_id=$2 AND deleted_at IS NULL
		UNION ALL
		SELECT id, 'face_audio' AS source, name AS event_type, 0 AS score, COALESCE(metadata,'{}'::jsonb), created_at
		FROM face_detection_logs
		WHERE tenant_id=$1 AND exam_session_id=$2 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 300`,
		tenantID, sessionID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	seen := map[uuid.UUID]bool{}
	for rows.Next() {
		var item ProctoringTimelineItem
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Source, &item.EventType, &item.Score, &raw, &item.CreatedAt); err != nil {
			return out, err
		}
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(raw, &item.Metadata)
		item.Severity, _ = item.Metadata["severity"].(string)
		if item.Severity == "" {
			item.Severity = severityFromScore(item.Score)
		}
		out.Summary[item.EventType]++
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}

func suspiciousScore(eventType, severity string) float64 {
	base := map[string]float64{
		"tab_switch":          10,
		"fullscreen_exit":     15,
		"copy_paste":          5,
		"right_click":         3,
		"devtools_suspected":  20,
		"multiple_monitor":    25,
		"abnormal_reconnect":  20,
		"device_changed":      40,
		"no_face":             10,
		"multiple_face":       25,
		"head_pose":           8,
		"audio_noise":         10,
		"webcam_unavailable":  30,
		"webcam_snapshot":     0,
		"face_detected":       0,
		"screen_recording_on": 30,
	}
	score := base[eventType]
	switch severity {
	case "critical":
		score *= 2
	case "high":
		score *= 1.5
	case "medium":
		score *= 1.2
	}
	if _, known := base[eventType]; score == 0 && !known {
		score = 1
	}
	return score
}

func (s *service) addSuspiciousScore(ctx context.Context, tenantID, sessionID uuid.UUID, score float64) float64 {
	if score <= 0 {
		return currentSuspiciousScore(ctx, s, tenantID, sessionID)
	}
	var raw []byte
	if err := s.deps.DB.QueryRow(ctx, `SELECT COALESCE(metadata,'{}'::jsonb) FROM exam_sessions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, sessionID, tenantID).Scan(&raw); err != nil {
		return score
	}
	metadata := map[string]any{}
	_ = json.Unmarshal(raw, &metadata)
	antiCheat := mapFromAny(metadata["anti_cheat"])
	total := numberFromAny(antiCheat["suspicious_score"]) + score
	antiCheat["suspicious_score"] = total
	switch {
	case total >= 90:
		antiCheat["status"] = "critical"
	case total >= 60:
		antiCheat["status"] = "suspicious"
	case total >= 30:
		antiCheat["status"] = "warning"
	default:
		antiCheat["status"] = "normal"
	}
	metadata["anti_cheat"] = antiCheat
	next, _ := json.Marshal(metadata)
	_, _ = s.deps.DB.Exec(ctx, `UPDATE exam_sessions SET metadata=$1, updated_at=now() WHERE id=$2 AND tenant_id=$3 AND deleted_at IS NULL`, next, sessionID, tenantID)
	return total
}

func currentSuspiciousScore(ctx context.Context, s *service, tenantID, sessionID uuid.UUID) float64 {
	var raw []byte
	if err := s.deps.DB.QueryRow(ctx, `SELECT COALESCE(metadata,'{}'::jsonb) FROM exam_sessions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, sessionID, tenantID).Scan(&raw); err != nil {
		return 0
	}
	metadata := map[string]any{}
	_ = json.Unmarshal(raw, &metadata)
	return numberFromAny(mapFromAny(metadata["anti_cheat"])["suspicious_score"])
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	out := map[string]any{}
	raw, err := json.Marshal(value)
	if err == nil {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func numberFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
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

func (s *service) ensureExamSessionAccess(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, sessionID uuid.UUID) error {
	if hasPermission(permissions, "*") || hasPermission(permissions, "proctoring:read") {
		var exists bool
		err := s.deps.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM exam_sessions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)`, sessionID, tenantID).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("exam session not found")
		}
		return nil
	}
	var allowed bool
	err := s.deps.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM exam_sessions es
			JOIN students st ON st.id=es.student_id AND st.deleted_at IS NULL
			WHERE es.id=$1 AND es.tenant_id=$2 AND st.user_id=$3 AND es.deleted_at IS NULL
		)`, sessionID, tenantID, actorUserID).Scan(&allowed)
	if err != nil || !allowed {
		return fmt.Errorf("exam session is not owned by current student")
	}
	return nil
}

func hasPermission(permissions []string, permission string) bool {
	for _, item := range permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func decodeSnapshot(input string) ([]byte, string, error) {
	value := strings.TrimSpace(input)
	contentType := "image/jpeg"
	if strings.HasPrefix(value, "data:") {
		parts := strings.SplitN(value, ",", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid snapshot data url")
		}
		header := strings.TrimPrefix(parts[0], "data:")
		header = strings.TrimSuffix(header, ";base64")
		if header != "" {
			contentType = header
		}
		value = parts[1]
	}
	payload, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, "", fmt.Errorf("invalid snapshot base64")
	}
	if len(payload) == 0 {
		return nil, "", fmt.Errorf("snapshot is empty")
	}
	detected := http.DetectContentType(payload)
	if detected == "image/jpeg" || detected == "image/png" {
		contentType = detected
	}
	if contentType != "image/jpeg" && contentType != "image/png" {
		return nil, "", fmt.Errorf("snapshot must be jpeg or png")
	}
	return payload, contentType, nil
}

func severityFromScore(score float64) string {
	switch {
	case score >= 18:
		return "critical"
	case score >= 10:
		return "high"
	case score >= 4:
		return "medium"
	default:
		return "low"
	}
}
