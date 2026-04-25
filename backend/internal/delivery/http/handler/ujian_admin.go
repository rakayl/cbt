package handler

import (
	"net/http"

	"cbt-enterprise/internal/domain"
	"cbt-enterprise/internal/infrastructure/websocket"
	"cbt-enterprise/internal/usecase"
	"cbt-enterprise/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── Ujian Handler ─────────────────────────────────────────────────────────────

type UjianHandler struct {
	uc *usecase.UjianUsecase
}

func NewUjianHandler(uc *usecase.UjianUsecase) *UjianHandler {
	return &UjianHandler{uc: uc}
}

func (h *UjianHandler) CreateUjian(c *gin.Context) {
	var req struct {
		Judul        string `json:"judul" binding:"required"`
		Deskripsi    string `json:"deskripsi"`
		MapelID      string `json:"mapel_id"`
		DurasiMenit  int    `json:"durasi_menit" binding:"required,min=1"`
		AcakSoal     bool   `json:"acak_soal"`
		AcakOpsi     bool   `json:"acak_opsi"`
		MaxTabSwitch int    `json:"max_tab_switch"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	guruID := c.MustGet("user_id").(uuid.UUID)
	mapelID, _ := uuid.Parse(req.MapelID)

	maxTab := req.MaxTabSwitch
	if maxTab == 0 {
		maxTab = 3
	}

	ujian, err := h.uc.CreateUjian(c.Request.Context(), usecase.CreateUjianInput{
		Judul:        req.Judul,
		Deskripsi:    req.Deskripsi,
		MapelID:      mapelID,
		GuruID:       guruID,
		DurasiMenit:  req.DurasiMenit,
		AcakSoal:     req.AcakSoal,
		AcakOpsi:     req.AcakOpsi,
		MaxTabSwitch: maxTab,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Created(c, ujian)
}

func (h *UjianHandler) ListUjian(c *gin.Context) {
	guruID := c.MustGet("user_id").(uuid.UUID)
	list, err := h.uc.ListByGuru(c.Request.Context(), guruID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, list)
}

func (h *UjianHandler) GetUjian(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	ujian, err := h.uc.GetUjian(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "ujian not found")
		return
	}
	response.Success(c, ujian)
}

func (h *UjianHandler) UpdateUjian(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	ujian, err := h.uc.GetUjian(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "ujian not found")
		return
	}
	if err := c.ShouldBindJSON(ujian); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.uc.UpdateUjian(c.Request.Context(), ujian); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, ujian)
}

func (h *UjianHandler) AddSoal(c *gin.Context) {
	ujianID, _ := uuid.Parse(c.Param("id"))
	var req struct {
		SoalID string `json:"soal_id" binding:"required"`
		Urutan int    `json:"urutan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	soalID, _ := uuid.Parse(req.SoalID)
	if err := h.uc.AddSoal(c.Request.Context(), ujianID, soalID, req.Urutan); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"added": true})
}

func (h *UjianHandler) AddPeserta(c *gin.Context) {
	ujianID, _ := uuid.Parse(c.Param("id"))
	var req struct {
		PesertaID string `json:"peserta_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	pesertaID, _ := uuid.Parse(req.PesertaID)
	if err := h.uc.AddPeserta(c.Request.Context(), ujianID, pesertaID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"added": true})
}

func (h *UjianHandler) SetStatus(c *gin.Context) {
	ujianID, _ := uuid.Parse(c.Param("id"))
	var req struct {
		Status string `json:"status" binding:"required,oneof=draft aktif selesai"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.uc.SetStatus(c.Request.Context(), ujianID, domain.StatusUjian(req.Status)); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": req.Status})
}

// ── Admin Handler ─────────────────────────────────────────────────────────────

type AdminHandler struct {
	hub           *websocket.Hub
	attemptRepo   domain.AttemptRepository
	ujianRepo     domain.UjianRepository
	penilaianRepo domain.PenilaianRepository
}

func NewAdminHandler(
	hub *websocket.Hub,
	ar domain.AttemptRepository,
	ur domain.UjianRepository,
	pr domain.PenilaianRepository,
) *AdminHandler {
	return &AdminHandler{hub: hub, attemptRepo: ar, ujianRepo: ur, penilaianRepo: pr}
}

// GET /api/v1/admin/online
func (h *AdminHandler) GetOnlinePeserta(c *gin.Context) {
	ids := h.hub.GetOnlineAttempts()
	response.Success(c, gin.H{
		"online_count": len(ids),
		"attempt_ids":  ids,
	})
}

// GET /api/v1/admin/ujian/:id/results
func (h *AdminHandler) GetResults(c *gin.Context) {
	ujianID, _ := uuid.Parse(c.Param("id"))
	attempts, err := h.attemptRepo.ListByUjian(c.Request.Context(), ujianID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, attempts)
}

// POST /api/v1/admin/attempt/:id/unpause
func (h *AdminHandler) UnpauseAttempt(c *gin.Context) {
	attemptID, _ := uuid.Parse(c.Param("id"))
	attempt, err := h.attemptRepo.FindByID(c.Request.Context(), attemptID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "attempt not found")
		return
	}
	attempt.Status = domain.AttemptOngoing
	if err := h.attemptRepo.Update(c.Request.Context(), attempt); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	// Notify peserta via WebSocket
	h.hub.SendToAttempt(attemptID.String(), &websocket.Message{
		Event:     "exam_resumed",
		AttemptID: attemptID.String(),
		Reason:    "admin_unpause",
	})
	response.Success(c, gin.H{"unpaused": true})
}
