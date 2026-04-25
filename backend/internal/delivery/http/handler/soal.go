package handler

import (
	"net/http"

	"cbt-enterprise/internal/domain"
	"cbt-enterprise/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SoalHandler struct {
	soalRepo domain.SoalRepository
}

func NewSoalHandler(sr domain.SoalRepository) *SoalHandler {
	return &SoalHandler{soalRepo: sr}
}

func (h *SoalHandler) CreateSoal(c *gin.Context) {
	var req struct {
		Pertanyaan string `json:"pertanyaan" binding:"required"`
		Tipe       string `json:"tipe" binding:"required,oneof=pilihan_ganda essay"`
		Poin       int    `json:"poin"`
		MapelID    string `json:"mapel_id"`
		Opsi       []struct {
			Teks    string `json:"teks"`
			IsBenar bool   `json:"is_benar"`
			Urutan  int    `json:"urutan"`
		} `json:"opsi"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	guruID := c.MustGet("user_id").(uuid.UUID)
	if req.Poin == 0 { req.Poin = 1 }

	soal := &domain.Soal{
		ID: uuid.New(), Pertanyaan: req.Pertanyaan,
		Tipe: domain.TipeSoal(req.Tipe), Poin: req.Poin, GuruID: guruID,
	}
	if req.MapelID != "" { soal.MapelID, _ = uuid.Parse(req.MapelID) }
	for _, o := range req.Opsi {
		soal.Opsi = append(soal.Opsi, domain.SoalOpsi{
			ID: uuid.New(), SoalID: soal.ID, Teks: o.Teks, IsBenar: o.IsBenar, Urutan: o.Urutan,
		})
	}
	if err := h.soalRepo.Create(c.Request.Context(), soal); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error()); return
	}
	response.Created(c, soal)
}

func (h *SoalHandler) ListSoal(c *gin.Context) {
	guruID := c.MustGet("user_id").(uuid.UUID)
	list, err := h.soalRepo.ListByGuru(c.Request.Context(), guruID)
	if err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	response.Success(c, list)
}

func (h *SoalHandler) GetSoal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil { response.Error(c, http.StatusBadRequest, "invalid id"); return }
	soal, err := h.soalRepo.FindByID(c.Request.Context(), id)
	if err != nil { response.Error(c, http.StatusNotFound, "soal not found"); return }
	response.Success(c, soal)
}

func (h *SoalHandler) DeleteSoal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil { response.Error(c, http.StatusBadRequest, "invalid id"); return }
	if err := h.soalRepo.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error()); return
	}
	response.Success(c, gin.H{"deleted": true})
}
