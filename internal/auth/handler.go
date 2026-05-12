package auth

import (
	"errors"

	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	user, err := h.service.Register(req.Username, req.Password, req.Name, req.Surname)
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			utils.Conflict(c, err.Error())
			return
		}
		utils.InternalError(c, "internal server error")
		return
	}

	utils.Created(c, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCreds) {
			utils.Unauthorized(c, err.Error())
			return
		}
		utils.InternalError(c, "internal server error")
		return
	}

	utils.Success(c, TokenResponse{Token: token})
}
