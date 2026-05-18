package video

import (
	"strings"

	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

// Service defines the contract that the video handler needs from the business layer.
// Method names use "Video" suffix to avoid collision with the legacy Service.Generate.
type Service interface {
	GenerateVideo(req *GenerateRequest) (*GenerateResponse, error)
	GetVideoStatus(taskID string) (*StatusResponse, error)
	CancelVideoTask(taskID string) error
	PreviewVideoPayload(req *GenerateRequest) (*PreviewPayloadResponse, error)
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Generate handles POST /studio/video/generate
func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.GenerateVideo(&req)
	if err != nil {
		if strings.Contains(err.Error(), "model not found") {
			utils.NotFound(c, err.Error())
			return
		}
		utils.BadRequest(c, err.Error())
		return
	}

	if len(result.Outputs) > 0 {
		utils.Created(c, result)
		return
	}

	utils.Created(c, result)
}

// GetStatus handles GET /studio/video/status/:taskId
func (h *Handler) GetStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "taskId is required")
		return
	}

	result, err := h.svc.GetVideoStatus(taskID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

// CancelTask handles DELETE /studio/video/task/:taskId
func (h *Handler) CancelTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "taskId is required")
		return
	}

	if err := h.svc.CancelVideoTask(taskID); err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Message(c, "task cancelled")
}

// PreviewPayload handles POST /studio/video/preview
func (h *Handler) PreviewPayload(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.PreviewVideoPayload(&req)
	if err != nil {
		if strings.Contains(err.Error(), "model not found") {
			utils.NotFound(c, err.Error())
			return
		}
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Success(c, result)
}
