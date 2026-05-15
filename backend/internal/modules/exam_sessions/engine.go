package exam_sessions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type StartExamRequest struct {
	ExamID            uuid.UUID `json:"exam_id" validate:"required"`
	StudentID         uuid.UUID `json:"student_id" validate:"required"`
	Token             string    `json:"token" validate:"required"`
	DeviceFingerprint string    `json:"device_fingerprint"`
	DeviceName        string    `json:"device_name"`
	UserAgent         string    `json:"user_agent"`
}

type AutosaveAnswerRequest struct {
	SessionID  uuid.UUID      `json:"session_id" validate:"required"`
	QuestionID uuid.UUID      `json:"question_id" validate:"required"`
	Payload    map[string]any `json:"payload" validate:"required"`
	ClientSeq  int64          `json:"client_seq"`
}

type ReconnectRequest struct {
	SessionID         uuid.UUID `json:"session_id" validate:"required"`
	LastClientSeq     int64     `json:"last_client_seq"`
	DeviceFingerprint string    `json:"device_fingerprint"`
	DeviceName        string    `json:"device_name"`
	UserAgent         string    `json:"user_agent"`
}

type ExamSessionState struct {
	SessionID         uuid.UUID `json:"session_id"`
	ExamID            uuid.UUID `json:"exam_id"`
	StudentID         uuid.UUID `json:"student_id"`
	Status            string    `json:"status"`
	StartedAt         time.Time `json:"started_at"`
	EndsAt            time.Time `json:"ends_at"`
	SubmittedAt       time.Time `json:"submitted_at,omitempty"`
	ServerTime        time.Time `json:"server_time"`
	RemainingSeconds  int64     `json:"remaining_seconds"`
	QuestionCount     int       `json:"question_count"`
	AutoSubmitted     bool      `json:"auto_submitted"`
	TimerMode         string    `json:"timer_mode"`
	TimerPaused       bool      `json:"timer_paused"`
	RecoveryStatus    string    `json:"recovery_status,omitempty"`
	ReviewRequired    bool      `json:"review_required"`
	ReconnectCount    int       `json:"reconnect_count"`
	TotalPauseSeconds int64     `json:"total_pause_seconds"`
	SuspiciousScore   float64   `json:"suspicious_score"`
}

func (s *service) StartExam(ctx context.Context, tenantID, userID uuid.UUID, req StartExamRequest) (ExamSessionState, error) {
	if err := validate.Struct(req); err != nil {
		return ExamSessionState{}, err
	}
	var ownStudentID uuid.UUID
	if err := s.deps.DB.QueryRow(ctx, `SELECT id FROM students WHERE tenant_id=$1 AND user_id=$2 AND deleted_at IS NULL LIMIT 1`, tenantID, userID).Scan(&ownStudentID); err == nil && ownStudentID != uuid.Nil {
		req.StudentID = ownStudentID
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return ExamSessionState{}, err
	}
	defer tx.Rollback(ctx)

	var examName string
	var examMetadataRaw []byte
	var durationMinutes, questionLimit int
	var randomQuestion, randomOption bool
	err = tx.QueryRow(ctx, `
		SELECT name,
		       duration_minutes,
		       random_question,
		       random_option,
		       CASE WHEN metadata->>'question_count' ~ '^[0-9]+$' THEN (metadata->>'question_count')::int ELSE 40 END,
		       COALESCE(metadata,'{}'::jsonb)
		FROM exams
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL AND status IN ('active','published')`,
		req.ExamID, tenantID).Scan(&examName, &durationMinutes, &randomQuestion, &randomOption, &questionLimit, &examMetadataRaw)
	if err != nil {
		return ExamSessionState{}, err
	}
	if questionLimit < 1 {
		questionLimit = 40
	}
	if questionLimit > 500 {
		questionLimit = 500
	}
	var inviteID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM exam_invites
		WHERE tenant_id=$1
		  AND exam_id=$2
		  AND student_id=$3
		  AND deleted_at IS NULL
		  AND lower(invitation_code)=lower($4)
		LIMIT 1`, tenantID, req.ExamID, req.StudentID, strings.TrimSpace(req.Token)).Scan(&inviteID)
	if err != nil {
		return ExamSessionState{}, errors.New("student is not invited to this exam or invitation code is invalid")
	}

	var existingSessionID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM exam_sessions
		WHERE tenant_id=$1
		  AND exam_id=$2
		  AND student_id=$3
		  AND deleted_at IS NULL
		  AND status_enum IN ('started','reconnecting')
		ORDER BY created_at DESC
		LIMIT 1`, tenantID, req.ExamID, req.StudentID).Scan(&existingSessionID)
	if err == nil && existingSessionID != uuid.Nil {
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return ExamSessionState{}, commitErr
		}
		return s.sessionState(ctx, tenantID, existingSessionID)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ExamSessionState{}, err
	}

	now := time.Now().UTC()
	endsAt := now.Add(time.Duration(durationMinutes) * time.Minute)
	sessionID := uuid.New()
	examMetadata := map[string]any{}
	_ = json.Unmarshal(examMetadataRaw, &examMetadata)
	recoveryPolicy := recoveryPolicyFromMetadata(examMetadata)
	antiCheatPolicy := antiCheatPolicyFromMetadata(examMetadata)
	sessionMetadata := initialSessionMetadata(recoveryPolicy, antiCheatPolicy, req, durationMinutes)
	rawSessionMetadata, _ := json.Marshal(sessionMetadata)
	_, err = tx.Exec(ctx, `
		INSERT INTO exam_sessions(id,tenant_id,code,name,status,status_enum,exam_id,student_id,started_at,ends_at,server_time_anchor,attempt,metadata)
		VALUES($1,$2,$3,$4,'started','started',$5,$6,$7,$8,$7,1,$9)`,
		sessionID, tenantID, "EXS-"+sessionID.String()[:8], examName, req.ExamID, req.StudentID, now, endsAt, rawSessionMetadata)
	if err != nil {
		return ExamSessionState{}, err
	}
	_, _ = tx.Exec(ctx, `UPDATE exam_invites SET accepted_at=COALESCE(accepted_at,now()), status='accepted', updated_at=now() WHERE id=$1 AND tenant_id=$2`, inviteID, tenantID)

	orderBy := "q.created_at ASC"
	if randomQuestion {
		orderBy = "random()"
	}
	questionIDs, err := s.selectExamQuestions(ctx, tx, tenantID, req.ExamID, questionLimit, orderBy)
	if err != nil {
		return ExamSessionState{}, err
	}

	position := 0
	for _, questionID := range questionIDs {
		position++
		optionOrder, err := s.optionOrder(ctx, questionID, randomOption)
		if err != nil {
			return ExamSessionState{}, err
		}
		snapshot, err := s.buildQuestionSnapshot(ctx, tx, tenantID, questionID, optionOrder)
		if err != nil {
			return ExamSessionState{}, err
		}
		rawOrder, _ := json.Marshal(optionOrder)
		rawMetadata, _ := json.Marshal(map[string]any{"snapshot": snapshot, "snapshot_version": 1})
		_, err = tx.Exec(ctx, `
			INSERT INTO exam_session_questions(id,tenant_id,code,name,status,metadata,exam_session_id,question_id,position,option_order)
			VALUES($1,$2,$3,$4,'active',$5,$6,$7,$8,$9)`,
			uuid.New(), tenantID, fmt.Sprintf("Q-%03d", position), fmt.Sprintf("Question %d", position), rawMetadata, sessionID, questionID, position, rawOrder)
		if err != nil {
			return ExamSessionState{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return ExamSessionState{}, err
	}

	state := ExamSessionState{SessionID: sessionID, ExamID: req.ExamID, StudentID: req.StudentID, Status: "started", StartedAt: now, EndsAt: endsAt, ServerTime: now, RemainingSeconds: int64(durationMinutes * 60), QuestionCount: position}
	applyRecoveryState(&state, sessionMetadata)
	if s.deps.Redis != nil {
		_ = s.writeTimerRedis(ctx, state)
		_ = s.deps.Redis.Expire(ctx, "exam_session:"+sessionID.String(), time.Duration(durationMinutes+60)*time.Minute).Err()
	}
	return state, nil
}

type examQuestionTagPool struct {
	QuestionTagID uuid.UUID
	QuestionCount int
}

type questionSnapshot struct {
	QuestionID   uuid.UUID         `json:"question_id"`
	Text         string            `json:"text"`
	QuestionType string            `json:"question_type"`
	AnswerMode   string            `json:"answer_mode"`
	Score        float64           `json:"score"`
	Media        snapshotMediaList `json:"media"`
	Options      []optionSnapshot  `json:"options"`
	CapturedAt   time.Time         `json:"captured_at"`
}

type optionSnapshot struct {
	ID        uuid.UUID         `json:"id"`
	Label     string            `json:"label"`
	Text      string            `json:"text"`
	IsCorrect bool              `json:"is_correct"`
	SortOrder int               `json:"sort_order"`
	Media     snapshotMediaList `json:"media"`
}

type snapshotMedia struct {
	ID        uuid.UUID `json:"id"`
	MimeType  string    `json:"mime_type"`
	FileSize  int64     `json:"file_size"`
	SortOrder int       `json:"sort_order"`
}

type snapshotMediaList []snapshotMedia

func (items snapshotMediaList) withURLs() []ExamMediaView {
	out := make([]ExamMediaView, 0, len(items))
	for _, item := range items {
		out = append(out, ExamMediaView{
			ID:        item.ID,
			URL:       fmt.Sprintf("/questions/media/%s/content", item.ID),
			MimeType:  item.MimeType,
			FileSize:  item.FileSize,
			SortOrder: item.SortOrder,
		})
	}
	return out
}

func (snapshot questionSnapshot) examOptions() []ExamOptionView {
	out := make([]ExamOptionView, 0, len(snapshot.Options))
	for _, option := range snapshot.Options {
		out = append(out, ExamOptionView{
			ID:    option.ID,
			Label: option.Label,
			Text:  option.Text,
			Media: option.Media.withURLs(),
		})
	}
	return out
}

func (snapshot questionSnapshot) correctOptionIDs() []uuid.UUID {
	out := []uuid.UUID{}
	for _, option := range snapshot.Options {
		if option.IsCorrect {
			out = append(out, option.ID)
		}
	}
	return out
}

func snapshotFromMetadata(raw []byte) (questionSnapshot, bool) {
	metadata := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return questionSnapshot{}, false
	}
	rawSnapshot, ok := metadata["snapshot"]
	if !ok || len(rawSnapshot) == 0 {
		return questionSnapshot{}, false
	}
	var snapshot questionSnapshot
	if err := json.Unmarshal(rawSnapshot, &snapshot); err != nil {
		return questionSnapshot{}, false
	}
	return snapshot, snapshot.QuestionID != uuid.Nil
}

func (s *service) buildQuestionSnapshot(ctx context.Context, tx pgx.Tx, tenantID, questionID uuid.UUID, optionOrder []uuid.UUID) (questionSnapshot, error) {
	var snapshot questionSnapshot
	err := tx.QueryRow(ctx, `
		SELECT id,
		       COALESCE(content, name, ''),
		       COALESCE(question_type, 'multiple_choice'),
		       COALESCE(answer_mode, 'single'),
		       CASE WHEN COALESCE(score,0) > 0 THEN score ELSE 1 END
		FROM questions
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, questionID, tenantID).
		Scan(&snapshot.QuestionID, &snapshot.Text, &snapshot.QuestionType, &snapshot.AnswerMode, &snapshot.Score)
	if err != nil {
		return questionSnapshot{}, err
	}
	snapshot.CapturedAt = time.Now().UTC()
	snapshot.Media, err = s.snapshotMedia(ctx, tx, tenantID, questionID, nil, "question")
	if err != nil {
		return questionSnapshot{}, err
	}
	options, err := s.snapshotOptions(ctx, tx, tenantID, questionID)
	if err != nil {
		return questionSnapshot{}, err
	}
	if len(optionOrder) > 0 {
		byID := map[uuid.UUID]optionSnapshot{}
		for _, option := range options {
			byID[option.ID] = option
		}
		ordered := []optionSnapshot{}
		used := map[uuid.UUID]bool{}
		for _, id := range optionOrder {
			if option, ok := byID[id]; ok {
				ordered = append(ordered, option)
				used[id] = true
			}
		}
		for _, option := range options {
			if !used[option.ID] {
				ordered = append(ordered, option)
			}
		}
		options = ordered
	}
	snapshot.Options = options
	return snapshot, nil
}

func (s *service) snapshotOptions(ctx context.Context, tx pgx.Tx, tenantID, questionID uuid.UUID) ([]optionSnapshot, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, code, COALESCE(description,''), is_correct, sort_order
		FROM question_options
		WHERE tenant_id=$1 AND question_id=$2 AND deleted_at IS NULL
		ORDER BY sort_order ASC`, tenantID, questionID)
	if err != nil {
		return nil, err
	}
	out := []optionSnapshot{}
	for rows.Next() {
		var item optionSnapshot
		if err := rows.Scan(&item.ID, &item.Label, &item.Text, &item.IsCorrect, &item.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, item)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return nil, rowsErr
	}
	for index := range out {
		out[index].Media, err = s.snapshotMedia(ctx, tx, tenantID, questionID, &out[index].ID, "option")
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *service) snapshotMedia(ctx context.Context, tx pgx.Tx, tenantID, questionID uuid.UUID, optionID *uuid.UUID, usageType string) (snapshotMediaList, error) {
	args := []any{tenantID, questionID, usageType}
	where := "tenant_id=$1 AND question_id=$2 AND usage_type=$3 AND deleted_at IS NULL"
	if optionID == nil {
		where += " AND option_id IS NULL"
	} else {
		where += fmt.Sprintf(" AND option_id=$%d", len(args)+1)
		args = append(args, *optionID)
	}
	rows, err := tx.Query(ctx, `
		SELECT id, mime_type, file_size, sort_order
		FROM question_media
		WHERE `+where+`
		ORDER BY sort_order ASC, created_at ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := snapshotMediaList{}
	for rows.Next() {
		var item snapshotMedia
		if err := rows.Scan(&item.ID, &item.MimeType, &item.FileSize, &item.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) selectExamQuestions(ctx context.Context, tx pgx.Tx, tenantID, examID uuid.UUID, questionLimit int, orderBy string) ([]uuid.UUID, error) {
	pools, err := s.examQuestionTagPools(ctx, tx, tenantID, examID)
	if err != nil {
		return nil, err
	}
	if len(pools) > 0 {
		return s.selectExamQuestionsByTagPools(ctx, tx, tenantID, pools, orderBy)
	}

	questionSQL := fmt.Sprintf(`
		SELECT q.id
		FROM questions q
		WHERE q.tenant_id=$1
		  AND q.deleted_at IS NULL
		  AND q.status='active'
		  AND (
		    q.question_bank_id IN (SELECT question_bank_id FROM exam_question_pools WHERE exam_id=$2 AND deleted_at IS NULL AND question_bank_id IS NOT NULL)
		    OR NOT EXISTS (SELECT 1 FROM exam_question_pools WHERE exam_id=$2 AND deleted_at IS NULL)
		  )
		ORDER BY %s
		LIMIT $3`, orderBy)
	rows, err := tx.Query(ctx, questionSQL, tenantID, examID, questionLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	questionIDs := []uuid.UUID{}
	for rows.Next() {
		var questionID uuid.UUID
		if err := rows.Scan(&questionID); err != nil {
			return nil, err
		}
		questionIDs = append(questionIDs, questionID)
	}
	return questionIDs, rows.Err()
}

func (s *service) examQuestionTagPools(ctx context.Context, tx pgx.Tx, tenantID, examID uuid.UUID) ([]examQuestionTagPool, error) {
	rows, err := tx.Query(ctx, `
		SELECT question_tag_id, question_count
		FROM exam_question_pools
		WHERE tenant_id=$1
		  AND exam_id=$2
		  AND deleted_at IS NULL
		  AND question_tag_id IS NOT NULL
		  AND question_count > 0
		ORDER BY created_at ASC`, tenantID, examID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pools := []examQuestionTagPool{}
	for rows.Next() {
		var pool examQuestionTagPool
		if err := rows.Scan(&pool.QuestionTagID, &pool.QuestionCount); err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}
	return pools, rows.Err()
}

func (s *service) selectExamQuestionsByTagPools(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, pools []examQuestionTagPool, orderBy string) ([]uuid.UUID, error) {
	questionIDs := []uuid.UUID{}
	used := map[uuid.UUID]bool{}
	for _, pool := range pools {
		rows, err := tx.Query(ctx, fmt.Sprintf(`
			SELECT q.id
			FROM questions q
			JOIN question_tag_relations qtr ON qtr.question_id=q.id AND qtr.deleted_at IS NULL
			WHERE q.tenant_id=$1
			  AND q.deleted_at IS NULL
			  AND q.status='active'
			  AND qtr.question_tag_id=$2
			ORDER BY %s`, orderBy), tenantID, pool.QuestionTagID)
		if err != nil {
			return nil, err
		}

		selectedForPool := 0
		for rows.Next() {
			var questionID uuid.UUID
			if err := rows.Scan(&questionID); err != nil {
				rows.Close()
				return nil, err
			}
			if used[questionID] {
				continue
			}
			used[questionID] = true
			questionIDs = append(questionIDs, questionID)
			selectedForPool++
			if selectedForPool >= pool.QuestionCount {
				break
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if selectedForPool < pool.QuestionCount {
			return nil, fmt.Errorf("question tag %s only has %d unique active questions available, need %d", pool.QuestionTagID, selectedForPool, pool.QuestionCount)
		}
	}
	return questionIDs, nil
}

func (s *service) sessionState(ctx context.Context, tenantID, sessionID uuid.UUID) (ExamSessionState, error) {
	state, err := s.autoSubmitIfExpired(ctx, tenantID, sessionID)
	if err != nil {
		return ExamSessionState{}, err
	}
	_ = s.deps.DB.QueryRow(ctx, `
		SELECT count(*)
		FROM exam_session_questions
		WHERE tenant_id=$1 AND exam_session_id=$2 AND deleted_at IS NULL`, tenantID, sessionID).Scan(&state.QuestionCount)
	if s.deps.Redis != nil {
		_ = s.writeTimerRedis(ctx, state)
		_ = s.deps.Redis.Expire(ctx, "exam_session:"+sessionID.String(), time.Duration(int64(state.RemainingSeconds)/60+60)*time.Minute).Err()
	}
	return state, nil
}

func (s *service) AutosaveAnswer(ctx context.Context, tenantID uuid.UUID, req AutosaveAnswerRequest) (map[string]any, error) {
	if err := validate.Struct(req); err != nil {
		return nil, err
	}
	state, err := s.autoSubmitIfExpired(ctx, tenantID, req.SessionID)
	if err != nil {
		return nil, err
	}
	if state.AutoSubmitted || state.Status == "completed" {
		return map[string]any{"saved": false, "status": "completed", "remaining_seconds": int64(0)}, nil
	}
	raw, _ := json.Marshal(req.Payload)
	_, err = s.deps.DB.Exec(ctx, `
		INSERT INTO answers(id,tenant_id,code,name,status,exam_session_id,question_id,answer_payload,autosaved_at,metadata)
		VALUES($1,$2,$3,'Autosaved answer','active',$4,$5,$6,now(),$7)
		ON CONFLICT (exam_session_id, question_id) WHERE deleted_at IS NULL
		DO UPDATE SET answer_payload=excluded.answer_payload, autosaved_at=now(), updated_at=now(), metadata=excluded.metadata`,
		uuid.New(), tenantID, "ANS-"+req.QuestionID.String()[:8], req.SessionID, req.QuestionID, raw, []byte(fmt.Sprintf(`{"client_seq":%d}`, req.ClientSeq)))
	if err != nil {
		return nil, err
	}
	if s.deps.Redis != nil {
		_ = s.deps.Redis.HSet(ctx, "autosave:"+req.SessionID.String(), req.QuestionID.String(), string(raw)).Err()
	}
	return map[string]any{"saved": true, "status": state.Status, "remaining_seconds": state.RemainingSeconds, "server_time": time.Now().UTC()}, nil
}

func (s *service) SessionQuestions(ctx context.Context, tenantID, userID, sessionID uuid.UUID) (ExamQuestionsResponse, error) {
	studentID, err := s.resolveStudentID(ctx, tenantID, userID)
	if err != nil {
		return ExamQuestionsResponse{}, err
	}
	state, err := s.autoSubmitIfExpired(ctx, tenantID, sessionID)
	if err != nil {
		return ExamQuestionsResponse{}, err
	}
	if state.StudentID != studentID {
		return ExamQuestionsResponse{}, errors.New("exam session belongs to another student")
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT esq.id, esq.question_id, esq.position, esq.code,
		       COALESCE(q.content, q.name, ''), COALESCE(q.question_type, 'multiple_choice'), COALESCE(q.answer_mode, 'single'),
		       esq.option_order, COALESCE(esq.metadata, '{}'::jsonb),
		       COALESCE(a.answer_payload, '{}'::jsonb)
		FROM exam_session_questions esq
		LEFT JOIN questions q ON q.id=esq.question_id
		LEFT JOIN answers a ON a.exam_session_id=esq.exam_session_id AND a.question_id=esq.question_id AND a.deleted_at IS NULL
		WHERE esq.tenant_id=$1 AND esq.exam_session_id=$2 AND esq.deleted_at IS NULL
		ORDER BY esq.position ASC`, tenantID, sessionID)
	if err != nil {
		return ExamQuestionsResponse{}, err
	}
	type questionRow struct {
		item        ExamQuestionView
		optionOrder []uuid.UUID
	}
	questionRows := []questionRow{}
	for rows.Next() {
		var qr questionRow
		var rawOrder, rawMetadata, rawAnswer []byte
		if err := rows.Scan(&qr.item.SessionQuestionID, &qr.item.QuestionID, &qr.item.Position, &qr.item.Code, &qr.item.Text, &qr.item.QuestionType, &qr.item.AnswerMode, &rawOrder, &rawMetadata, &rawAnswer); err != nil {
			rows.Close()
			return ExamQuestionsResponse{}, err
		}
		_ = json.Unmarshal(rawOrder, &qr.optionOrder)
		_ = json.Unmarshal(rawAnswer, &qr.item.AnswerPayload)
		if snapshot, ok := snapshotFromMetadata(rawMetadata); ok {
			qr.item.QuestionID = snapshot.QuestionID
			qr.item.Text = snapshot.Text
			qr.item.QuestionType = snapshot.QuestionType
			qr.item.AnswerMode = snapshot.AnswerMode
			qr.item.Media = snapshot.Media.withURLs()
			qr.item.Options = snapshot.examOptions()
		}
		questionRows = append(questionRows, qr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return ExamQuestionsResponse{}, err
	}
	out := ExamQuestionsResponse{
		SessionID:         sessionID,
		Status:            state.Status,
		StartedAt:         state.StartedAt,
		EndsAt:            state.EndsAt,
		ServerTime:        state.ServerTime,
		RemainingSeconds:  state.RemainingSeconds,
		TimerMode:         state.TimerMode,
		TimerPaused:       state.TimerPaused,
		RecoveryStatus:    state.RecoveryStatus,
		ReviewRequired:    state.ReviewRequired,
		ReconnectCount:    state.ReconnectCount,
		TotalPauseSeconds: state.TotalPauseSeconds,
		SuspiciousScore:   state.SuspiciousScore,
		Questions:         make([]ExamQuestionView, 0, len(questionRows)),
	}
	for _, qr := range questionRows {
		if len(qr.item.Options) > 0 {
			out.Questions = append(out.Questions, qr.item)
			continue
		}
		media, err := s.examMedia(ctx, tenantID, qr.item.QuestionID, nil, "question")
		if err != nil {
			return ExamQuestionsResponse{}, err
		}
		qr.item.Media = media
		options, err := s.examOptions(ctx, qr.item.QuestionID, qr.optionOrder)
		if err != nil {
			return ExamQuestionsResponse{}, err
		}
		qr.item.Options = options
		out.Questions = append(out.Questions, qr.item)
	}
	return out, nil
}

func (s *service) examOptions(ctx context.Context, questionID uuid.UUID, order []uuid.UUID) ([]ExamOptionView, error) {
	rows, err := s.deps.DB.Query(ctx, `
		SELECT id, code, description
		FROM question_options
		WHERE question_id=$1 AND deleted_at IS NULL
		ORDER BY sort_order ASC`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[uuid.UUID]ExamOptionView{}
	fallback := []ExamOptionView{}
	for rows.Next() {
		var option ExamOptionView
		if err := rows.Scan(&option.ID, &option.Label, &option.Text); err != nil {
			return nil, err
		}
		media, err := s.examMedia(ctx, uuid.Nil, questionID, &option.ID, "option")
		if err != nil {
			return nil, err
		}
		option.Media = media
		byID[option.ID] = option
		fallback = append(fallback, option)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return fallback, nil
	}
	out := []ExamOptionView{}
	used := map[uuid.UUID]bool{}
	for _, id := range order {
		if option, ok := byID[id]; ok {
			out = append(out, option)
			used[id] = true
		}
	}
	for _, option := range fallback {
		if !used[option.ID] {
			out = append(out, option)
		}
	}
	return out, nil
}

func (s *service) examMedia(ctx context.Context, tenantID, questionID uuid.UUID, optionID *uuid.UUID, usageType string) ([]ExamMediaView, error) {
	args := []any{questionID, usageType}
	where := "question_id=$1 AND usage_type=$2 AND deleted_at IS NULL"
	if tenantID != uuid.Nil {
		where += " AND tenant_id=$3"
		args = append(args, tenantID)
	}
	if optionID == nil {
		where += " AND option_id IS NULL"
	} else {
		where += fmt.Sprintf(" AND option_id=$%d", len(args)+1)
		args = append(args, *optionID)
	}
	rows, err := s.deps.DB.Query(ctx, `
		SELECT id, mime_type, file_size, sort_order
		FROM question_media
		WHERE `+where+`
		ORDER BY sort_order ASC, created_at ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ExamMediaView{}
	for rows.Next() {
		var item ExamMediaView
		if err := rows.Scan(&item.ID, &item.MimeType, &item.FileSize, &item.SortOrder); err != nil {
			return nil, err
		}
		item.URL = fmt.Sprintf("/questions/media/%s/content", item.ID)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) Reconnect(ctx context.Context, tenantID, userID uuid.UUID, req ReconnectRequest) (ExamSessionState, error) {
	if err := validate.Struct(req); err != nil {
		return ExamSessionState{}, err
	}
	studentID, err := s.resolveStudentID(ctx, tenantID, userID)
	if err != nil {
		return ExamSessionState{}, err
	}
	var ownerStudentID uuid.UUID
	if err := s.deps.DB.QueryRow(ctx, `SELECT student_id FROM exam_sessions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, req.SessionID, tenantID).Scan(&ownerStudentID); err != nil {
		return ExamSessionState{}, err
	}
	if ownerStudentID != studentID {
		return ExamSessionState{}, errors.New("exam session belongs to another student")
	}
	if err := s.resumeRecoveryTimer(ctx, tenantID, req); err != nil {
		return ExamSessionState{}, err
	}
	state, err := s.autoSubmitIfExpired(ctx, tenantID, req.SessionID)
	if err != nil {
		return ExamSessionState{}, err
	}
	if s.deps.Redis != nil {
		_ = s.writeTimerRedis(ctx, state)
	}
	return state, nil
}

func (s *service) SubmitExam(ctx context.Context, tenantID, userID, sessionID uuid.UUID, req SubmitExamRequest) (SubmitExamResponse, error) {
	studentID, err := s.resolveStudentID(ctx, tenantID, userID)
	if err != nil {
		return SubmitExamResponse{}, err
	}
	var ownerStudentID uuid.UUID
	if err := s.deps.DB.QueryRow(ctx, `SELECT student_id FROM exam_sessions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, sessionID, tenantID).Scan(&ownerStudentID); err != nil {
		return SubmitExamResponse{}, err
	}
	if ownerStudentID != studentID {
		return SubmitExamResponse{}, errors.New("exam session belongs to another student")
	}
	return s.submitSession(ctx, tenantID, sessionID, false, req.ClientSeq)
}

func (s *service) AutoSubmitExpiredSessions(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	rows, err := s.deps.DB.Query(ctx, `
		SELECT id
		FROM exam_sessions
		WHERE tenant_id=$1
		  AND deleted_at IS NULL
		  AND ends_at <= now()
		  AND status_enum IN ('started','reconnecting')
		  AND NOT (status_enum='reconnecting' AND metadata->'recovery'->>'timer_paused'='true')`, tenantID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	sessionIDs := []uuid.UUID{}
	for rows.Next() {
		var sessionID uuid.UUID
		if err := rows.Scan(&sessionID); err != nil {
			return 0, err
		}
		sessionIDs = append(sessionIDs, sessionID)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var submitted int64
	for _, sessionID := range sessionIDs {
		if _, err := s.submitSession(ctx, tenantID, sessionID, true, 0); err == nil {
			submitted++
		} else {
			return submitted, err
		}
	}
	return submitted, nil
}

func (s *service) autoSubmitIfExpired(ctx context.Context, tenantID, sessionID uuid.UUID) (ExamSessionState, error) {
	var state ExamSessionState
	var statusEnum string
	var rawMetadata []byte
	err := s.deps.DB.QueryRow(ctx, `
		SELECT id, exam_id, student_id, status_enum::text, started_at, ends_at, COALESCE(metadata,'{}'::jsonb)
		FROM exam_sessions
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		sessionID, tenantID).Scan(&state.SessionID, &state.ExamID, &state.StudentID, &statusEnum, &state.StartedAt, &state.EndsAt, &rawMetadata)
	if err != nil {
		return state, err
	}
	metadata := map[string]any{}
	_ = json.Unmarshal(rawMetadata, &metadata)
	applyRecoveryState(&state, metadata)
	now := time.Now().UTC()
	state.ServerTime = now
	state.Status = statusEnum
	if state.TimerPaused && statusEnum == "reconnecting" {
		state.RemainingSeconds = int64(math.Max(0, float64(state.RemainingSeconds)))
		return state, nil
	}
	state.RemainingSeconds = int64(math.Max(0, state.EndsAt.Sub(now).Seconds()))
	if state.RemainingSeconds == 0 && (statusEnum == "started" || statusEnum == "reconnecting") {
		summary, err := s.submitSession(ctx, tenantID, sessionID, true, 0)
		if err != nil {
			return state, err
		}
		state.Status = "completed"
		state.RemainingSeconds = 0
		state.ServerTime = time.Now().UTC()
		state.SubmittedAt = summary.SubmittedAt
		state.AutoSubmitted = true
	}
	return state, nil
}

func (s *service) submitSession(ctx context.Context, tenantID, sessionID uuid.UUID, auto bool, clientSeq int64) (SubmitExamResponse, error) {
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return SubmitExamResponse{}, err
	}
	defer tx.Rollback(ctx)

	var status string
	var passingGrade float64
	var submittedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT es.status_enum::text, es.submitted_at, COALESCE(e.passing_grade,60)
		FROM exam_sessions es
		JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
		WHERE es.id=$1 AND es.tenant_id=$2 AND es.deleted_at IS NULL
		FOR UPDATE OF es`, sessionID, tenantID).Scan(&status, &submittedAt, &passingGrade)
	if err != nil {
		return SubmitExamResponse{}, err
	}
	if status == "completed" {
		if submittedAt == nil {
			now := time.Now().UTC()
			submittedAt = &now
		}
		out, err := s.existingSubmitSummary(ctx, tx, tenantID, sessionID, *submittedAt, passingGrade)
		if err != nil {
			return SubmitExamResponse{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return SubmitExamResponse{}, err
		}
		return out, nil
	}
	if status != "started" && status != "reconnecting" {
		return SubmitExamResponse{}, fmt.Errorf("session status %s cannot be submitted", status)
	}

	summary, err := s.gradeSession(ctx, tx, tenantID, sessionID, passingGrade)
	if err != nil {
		return SubmitExamResponse{}, err
	}
	now := time.Now().UTC()
	summary.SessionID = sessionID
	summary.Status = "completed"
	summary.SubmittedAt = now
	meta := map[string]any{
		"score":            summary.Score,
		"max_score":        summary.MaxScore,
		"percentage":       summary.Percentage,
		"passing_grade":    summary.PassingGrade,
		"passed":           summary.Passed,
		"correct_count":    summary.CorrectCount,
		"wrong_count":      summary.WrongCount,
		"answered_count":   summary.AnsweredCount,
		"unanswered_count": summary.UnansweredCount,
		"auto_submitted":   auto,
		"client_seq":       clientSeq,
		"graded_at":        now.Format(time.RFC3339Nano),
	}
	meta["result_integrity_hash"] = resultIntegrityHash(sessionID, summary, auto, clientSeq)
	rawMeta, _ := json.Marshal(meta)
	err = tx.QueryRow(ctx, `
		UPDATE exam_sessions
		SET status='completed',
		    status_enum='completed',
		    submitted_at=COALESCE(submitted_at, $1),
		    metadata=COALESCE(metadata,'{}'::jsonb) || $2::jsonb,
		    updated_at=now()
		WHERE id=$3 AND tenant_id=$4 AND deleted_at IS NULL
		RETURNING submitted_at`, now, rawMeta, sessionID, tenantID).Scan(&summary.SubmittedAt)
	if err != nil {
		return SubmitExamResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SubmitExamResponse{}, err
	}
	if s.deps.Redis != nil {
		_ = s.deps.Redis.HSet(ctx, "exam_session:"+sessionID.String(), "status", "completed", "remaining_seconds", 0, "submitted_at", summary.SubmittedAt.Format(time.RFC3339Nano)).Err()
	}
	if s.deps.Rabbit != nil {
		body, _ := json.Marshal(summary)
		_ = s.deps.Rabbit.Publish(ctx, "grading_queue", body)
	}
	return summary, nil
}

func (s *service) existingSubmitSummary(ctx context.Context, tx pgx.Tx, tenantID, sessionID uuid.UUID, submittedAt time.Time, passingGrade float64) (SubmitExamResponse, error) {
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT metadata FROM exam_sessions WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`, sessionID, tenantID).Scan(&raw); err != nil {
		return SubmitExamResponse{}, err
	}
	meta := map[string]any{}
	_ = json.Unmarshal(raw, &meta)
	out := SubmitExamResponse{SessionID: sessionID, Status: "completed", SubmittedAt: submittedAt, PassingGrade: passingGrade}
	out.Score = numberFromMeta(meta, "score")
	out.MaxScore = numberFromMeta(meta, "max_score")
	out.Percentage = numberFromMeta(meta, "percentage")
	if out.PassingGrade == 0 {
		out.PassingGrade = numberFromMeta(meta, "passing_grade")
	}
	out.Passed, _ = meta["passed"].(bool)
	out.CorrectCount = int(numberFromMeta(meta, "correct_count"))
	out.WrongCount = int(numberFromMeta(meta, "wrong_count"))
	out.AnsweredCount = int(numberFromMeta(meta, "answered_count"))
	out.UnansweredCount = int(numberFromMeta(meta, "unanswered_count"))
	return out, nil
}

func resultIntegrityHash(sessionID uuid.UUID, summary SubmitExamResponse, auto bool, clientSeq int64) string {
	payload := fmt.Sprintf(
		"%s|%.4f|%.4f|%.4f|%.4f|%t|%d|%d|%d|%d|%t|%d",
		sessionID.String(),
		summary.Score,
		summary.MaxScore,
		summary.Percentage,
		summary.PassingGrade,
		summary.Passed,
		summary.CorrectCount,
		summary.WrongCount,
		summary.AnsweredCount,
		summary.UnansweredCount,
		auto,
		clientSeq,
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func numberFromMeta(meta map[string]any, key string) float64 {
	switch value := meta[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case json.Number:
		out, _ := value.Float64()
		return out
	default:
		return 0
	}
}

type gradingQuestionRow struct {
	SessionQuestionID uuid.UUID
	QuestionID        uuid.UUID
	AnswerMode        string
	MaxScore          float64
	AnswerPayload     map[string]any
	CorrectOptionIDs  []uuid.UUID
}

func (s *service) gradeSession(ctx context.Context, tx pgx.Tx, tenantID, sessionID uuid.UUID, passingGrade float64) (SubmitExamResponse, error) {
	rows, err := tx.Query(ctx, `
		SELECT esq.id,
		       esq.question_id,
		       COALESCE(esq.metadata, '{}'::jsonb),
		       COALESCE(q.answer_mode, 'single'),
		       CASE WHEN COALESCE(q.score,0) > 0 THEN q.score ELSE 1 END,
		       COALESCE(a.answer_payload, '{}'::jsonb),
		       COALESCE((
		         SELECT jsonb_agg(qo.id::text ORDER BY qo.id::text)
		         FROM question_options qo
		         WHERE qo.question_id=esq.question_id AND qo.is_correct
		       ), '[]'::jsonb)
		FROM exam_session_questions esq
		LEFT JOIN questions q ON q.id=esq.question_id
		LEFT JOIN answers a ON a.exam_session_id=esq.exam_session_id AND a.question_id=esq.question_id AND a.deleted_at IS NULL
		WHERE esq.tenant_id=$1 AND esq.exam_session_id=$2 AND esq.deleted_at IS NULL
		ORDER BY esq.position ASC`, tenantID, sessionID)
	if err != nil {
		return SubmitExamResponse{}, err
	}

	items := []gradingQuestionRow{}
	for rows.Next() {
		var item gradingQuestionRow
		var rawMetadata, rawAnswer, rawCorrect []byte
		if err := rows.Scan(&item.SessionQuestionID, &item.QuestionID, &rawMetadata, &item.AnswerMode, &item.MaxScore, &rawAnswer, &rawCorrect); err != nil {
			return SubmitExamResponse{}, err
		}
		_ = json.Unmarshal(rawAnswer, &item.AnswerPayload)
		item.CorrectOptionIDs = uuidSliceFromJSON(rawCorrect)
		if snapshot, ok := snapshotFromMetadata(rawMetadata); ok {
			item.QuestionID = snapshot.QuestionID
			item.AnswerMode = snapshot.AnswerMode
			item.MaxScore = snapshot.Score
			if item.MaxScore <= 0 {
				item.MaxScore = 1
			}
			item.CorrectOptionIDs = snapshot.correctOptionIDs()
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return SubmitExamResponse{}, err
	}
	rows.Close()

	_, err = tx.Exec(ctx, `
		UPDATE grading
		SET deleted_at=now(), updated_at=now()
		WHERE tenant_id=$1 AND deleted_at IS NULL AND metadata->>'exam_session_id'=$2`, tenantID, sessionID.String())
	if err != nil {
		return SubmitExamResponse{}, err
	}

	out := SubmitExamResponse{SessionID: sessionID, Status: "completed", PassingGrade: passingGrade}
	for index, item := range items {
		selected := uuidSliceFromAny(item.AnswerPayload["selected_option_ids"])
		answered := len(selected) > 0 || strings.TrimSpace(fmt.Sprint(item.AnswerPayload["text"])) != "" && fmt.Sprint(item.AnswerPayload["text"]) != "<nil>"
		isCorrect := false
		if item.AnswerMode == "single" || item.AnswerMode == "multiple" {
			isCorrect = sameUUIDSet(selected, item.CorrectOptionIDs)
		}
		earned := 0.0
		if isCorrect {
			earned = item.MaxScore
			out.CorrectCount++
		} else if answered {
			out.WrongCount++
		}
		if answered {
			out.AnsweredCount++
		} else {
			out.UnansweredCount++
		}
		out.Score += earned
		out.MaxScore += item.MaxScore
		meta := map[string]any{
			"exam_session_id":     sessionID.String(),
			"session_question_id": item.SessionQuestionID.String(),
			"question_id":         item.QuestionID.String(),
			"answer_mode":         item.AnswerMode,
			"selected_option_ids": uuidStrings(selected),
			"correct_option_ids":  uuidStrings(item.CorrectOptionIDs),
			"is_correct":          isCorrect,
			"earned_score":        earned,
			"max_score":           item.MaxScore,
			"answered":            answered,
		}
		rawMeta, _ := json.Marshal(meta)
		_, err := tx.Exec(ctx, `
			INSERT INTO grading(id,tenant_id,code,name,status,metadata)
			VALUES($1,$2,$3,$4,'completed',$5)`,
			uuid.New(), tenantID, fmt.Sprintf("GRD-%s-%03d", sessionID.String()[:8], index+1), fmt.Sprintf("Question %d grading", index+1), rawMeta)
		if err != nil {
			return SubmitExamResponse{}, err
		}
	}
	if out.MaxScore > 0 {
		out.Percentage = math.Round((out.Score/out.MaxScore)*10000) / 100
	}
	out.Passed = out.Percentage >= passingGrade
	return out, nil
}

func uuidSliceFromJSON(raw []byte) []uuid.UUID {
	values := []string{}
	_ = json.Unmarshal(raw, &values)
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if id, err := uuid.Parse(value); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func uuidSliceFromAny(value any) []uuid.UUID {
	out := []uuid.UUID{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if id, err := uuid.Parse(fmt.Sprint(item)); err == nil {
				out = append(out, id)
			}
		}
	case []string:
		for _, item := range typed {
			if id, err := uuid.Parse(item); err == nil {
				out = append(out, id)
			}
		}
	case string:
		if id, err := uuid.Parse(typed); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func sameUUIDSet(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	left := uuidStrings(a)
	right := uuidStrings(b)
	sort.Strings(left)
	sort.Strings(right)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func uuidStrings(values []uuid.UUID) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.String())
	}
	return out
}

func (s *service) optionOrder(ctx context.Context, questionID uuid.UUID, random bool) ([]uuid.UUID, error) {
	orderBy := "created_at ASC"
	if random {
		orderBy = "random()"
	}
	rows, err := s.deps.DB.Query(ctx, fmt.Sprintf(`SELECT id FROM question_options WHERE question_id=$1 AND deleted_at IS NULL ORDER BY %s`, orderBy), questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
