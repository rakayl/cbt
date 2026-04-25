package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"cbt-enterprise/internal/infrastructure/ai"
	"cbt-enterprise/internal/infrastructure/websocket"
	"cbt-enterprise/internal/usecase"
	"cbt-enterprise/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
)

var upgrader = gorillaws.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ─── ATTEMPT HANDLER ─────────────────────────────────────────────────────────

type AttemptHandler struct {
	uc         *usecase.AttemptUsecase
	hub        *websocket.Hub
	aiClient   *ai.ProctoringClient
}

func NewAttemptHandler(uc *usecase.AttemptUsecase, hub *websocket.Hub, aiClient *ai.ProctoringClient) *AttemptHandler {
	return &AttemptHandler{uc: uc, hub: hub, aiClient: aiClient}
}

// POST /api/v1/exam/start
func (h *AttemptHandler) StartExam(c *gin.Context) {
	var req struct {
		UjianID string `json:"ujian_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	pesertaID := c.MustGet("user_id").(uuid.UUID)
	ujianID, err := uuid.Parse(req.UjianID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid ujian_id")
		return
	}

	attempt, soalList, err := h.uc.StartExam(c.Request.Context(), pesertaID, ujianID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, gin.H{
		"attempt":   attempt,
		"soal_list": soalList,
	})
}

// POST /api/v1/exam/answer
func (h *AttemptHandler) SaveAnswer(c *gin.Context) {
	var req struct {
		AttemptID   string  `json:"attempt_id" binding:"required"`
		SoalID      string  `json:"soal_id" binding:"required"`
		OpsiID      *string `json:"opsi_id"`
		TeksJawaban string  `json:"teks_jawaban"`
		SisaDetik   int     `json:"sisa_detik"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	attemptID, _ := uuid.Parse(req.AttemptID)
	soalID, _ := uuid.Parse(req.SoalID)

	var opsiID *uuid.UUID
	if req.OpsiID != nil {
		id, _ := uuid.Parse(*req.OpsiID)
		opsiID = &id
	}

	input := usecase.SaveJawabanInput{
		AttemptID:   attemptID,
		SoalID:      soalID,
		OpsiID:      opsiID,
		TeksJawaban: req.TeksJawaban,
		SisaDetik:   req.SisaDetik,
	}

	if err := h.uc.SaveJawaban(c.Request.Context(), input); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"saved": true})
}

// POST /api/v1/exam/finish
func (h *AttemptHandler) FinishExam(c *gin.Context) {
	var req struct {
		AttemptID string `json:"attempt_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	attemptID, _ := uuid.Parse(req.AttemptID)
	if err := h.uc.FinishExam(c.Request.Context(), attemptID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Notify via WebSocket
	h.hub.SendToAttempt(req.AttemptID, &websocket.Message{
		Event:     websocket.EventExamFinished,
		AttemptID: req.AttemptID,
	})

	response.Success(c, gin.H{"finished": true})
}

// POST /api/v1/exam/violation
func (h *AttemptHandler) ReportViolation(c *gin.Context) {
	var req struct {
		AttemptID string `json:"attempt_id" binding:"required"`
		EventType string `json:"event_type" binding:"required"`
		Detail    string `json:"detail"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	pesertaID := c.MustGet("user_id").(uuid.UUID)
	attemptID, _ := uuid.Parse(req.AttemptID)

	paused, err := h.uc.LogViolation(c.Request.Context(), usecase.ViolationInput{
		AttemptID: attemptID,
		PesertaID: pesertaID,
		EventType: req.EventType,
		Detail:    req.Detail,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if paused {
		// Push exam_paused to peserta via WS
		h.hub.SendToAttempt(req.AttemptID, &websocket.Message{
			Event:     websocket.EventExamPaused,
			AttemptID: req.AttemptID,
			Reason:    "cheating_detected",
		})
		// Alert admins
		h.hub.BroadcastToAdmins(&websocket.Message{
			Event:     websocket.EventCheatingDetected,
			AttemptID: req.AttemptID,
			Payload: map[string]string{
				"reason": req.EventType,
				"detail": req.Detail,
			},
		})
	}

	response.Success(c, gin.H{"paused": paused})
}

// POST /api/v1/exam/proctor
func (h *AttemptHandler) ProcessFrame(c *gin.Context) {
	var req struct {
		AttemptID   string  `json:"attempt_id" binding:"required"`
		ImageBase64 string  `json:"image_base64" binding:"required"`
		BaseEmbedding []float64 `json:"base_embedding"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()

	// Detect face count
	detect, err := h.aiClient.DetectFace(ctx, ai.DetectRequest{
		ImageBase64: req.ImageBase64,
		AttemptID:   req.AttemptID,
	})
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "proctoring service unavailable")
		return
	}

	violation := ""
	if !detect.HasFace {
		violation = "no_face"
	} else if detect.FaceCount > 1 {
		violation = "multiple_faces"
	}

	// Face verification against baseline
	if detect.HasFace && detect.FaceCount == 1 && len(req.BaseEmbedding) > 0 {
		verify, err := h.aiClient.VerifyFace(ctx, ai.VerifyRequest{
			ImageBase64:   req.ImageBase64,
			BaseEmbedding: req.BaseEmbedding,
			AttemptID:     req.AttemptID,
		})
		if err == nil && !verify.Match {
			violation = "face_mismatch"
		}
	}

	if violation != "" {
		pesertaID := c.MustGet("user_id").(uuid.UUID)
		attemptID, _ := uuid.Parse(req.AttemptID)

		paused, _ := h.uc.LogViolation(ctx, usecase.ViolationInput{
			AttemptID: attemptID,
			PesertaID: pesertaID,
			EventType: violation,
			Detail:    "AI proctoring detection",
		})

		if paused {
			h.hub.SendToAttempt(req.AttemptID, &websocket.Message{
				Event:     websocket.EventExamPaused,
				AttemptID: req.AttemptID,
				Reason:    "cheating_detected",
			})
		}

		h.hub.BroadcastToAdmins(&websocket.Message{
			Event:     websocket.EventFaceAlert,
			AttemptID: req.AttemptID,
			Payload: map[string]interface{}{
				"violation": violation,
				"face_count": detect.FaceCount,
			},
		})
	}

	response.Success(c, gin.H{
		"violation": violation,
		"face_count": detect.FaceCount,
	})
}

// GET /api/v1/exam/attempt/:id
func (h *AttemptHandler) GetAttemptState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	attempt, jawabans, err := h.uc.GetAttemptState(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "attempt not found")
		return
	}

	response.Success(c, gin.H{
		"attempt":  attempt,
		"jawabans": jawabans,
	})
}

// ─── WEBSOCKET HANDLER ────────────────────────────────────────────────────────

type WSHandler struct {
	hub *websocket.Hub
}

func NewWSHandler(hub *websocket.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// GET /ws
func (h *WSHandler) ServeWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role").(string)
	attemptID := c.Query("attempt_id")

	client := &websocket.Client{
		ID:        userID.String(),
		Role:      role,
		AttemptID: attemptID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		Hub:       h.hub,
	}

	h.hub.Register <- client

	// Send heartbeat confirmation
	hello, _ := json.Marshal(websocket.Message{
		Event:   websocket.EventHeartbeat,
		Payload: map[string]interface{}{"ts": time.Now().Unix(), "connected": true},
	})
	client.Send <- hello

	go client.WritePump()
	client.ReadPump()
}
