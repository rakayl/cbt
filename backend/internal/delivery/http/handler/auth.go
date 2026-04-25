package handler

import (
	"net/http"

	"cbt-enterprise/internal/domain"
	"cbt-enterprise/internal/usecase"
	"cbt-enterprise/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	uc *usecase.AuthUsecase
}

func NewAuthHandler(uc *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Nama     string `json:"nama" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Role     string `json:"role" binding:"required,oneof=admin guru peserta"`
		NIS      string `json:"nis"`
		Kelas    string `json:"kelas"`
		NIP      string `json:"nip"`
		Mapel    string `json:"mapel"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.uc.Register(c.Request.Context(), usecase.RegisterInput{
		Nama:     req.Nama,
		Email:    req.Email,
		Password: req.Password,
		Role:     domain.Role(req.Role),
		NIS:      req.NIS,
		Kelas:    req.Kelas,
		NIP:      req.NIP,
		Mapel:    req.Mapel,
	})
	if err != nil {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}

	response.Success(c, gin.H{
		"token": result.Token,
		"user": gin.H{
			"id":    result.User.ID,
			"nama":  result.User.Nama,
			"email": result.User.Email,
			"role":  result.User.Role,
		},
	})
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.uc.Login(c.Request.Context(), usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	response.Success(c, gin.H{
		"token": result.Token,
		"user": gin.H{
			"id":    result.User.ID,
			"nama":  result.User.Nama,
			"email": result.User.Email,
			"role":  result.User.Role,
		},
	})
}
