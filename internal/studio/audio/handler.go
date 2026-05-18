package audio

import (
	"fmt"

	"dcs-back-v0/internal/studio"
	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *studio.Service
}

func NewHandler(svc *studio.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Generate(c *gin.Context) {
	utils.BadRequest(c, fmt.Sprintf("audio generation not yet implemented"))
}

func (h *Handler) GetStatus(c *gin.Context) {
	utils.BadRequest(c, fmt.Sprintf("audio generation not yet implemented"))
}

func (h *Handler) CancelTask(c *gin.Context) {
	utils.BadRequest(c, fmt.Sprintf("audio generation not yet implemented"))
}

func (h *Handler) PreviewPayload(c *gin.Context) {
	utils.BadRequest(c, fmt.Sprintf("audio generation not yet implemented"))
}
