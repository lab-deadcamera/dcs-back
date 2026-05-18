package image

import (
	"strings"

	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

// Service defines the contract that the image handler needs from the business layer.
// Method names use "Image" suffix to avoid collision with the legacy Service.Generate.
type Service interface {
	GenerateImage(req *GenerateRequest) (*GenerateResponse, error)
	GetImageStatus(taskID string) (*StatusResponse, error)
	CancelImageTask(taskID string) error
	PreviewImagePayload(req *GenerateRequest) (*PreviewPayloadResponse, error)
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Generate handles POST /studio/image/generate
func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.GenerateImage(&req)
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

// GetStatus handles GET /studio/image/status/:taskId
func (h *Handler) GetStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "taskId is required")
		return
	}

	result, err := h.svc.GetImageStatus(taskID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

// CancelTask handles DELETE /studio/image/task/:taskId
func (h *Handler) CancelTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "taskId is required")
		return
	}

	if err := h.svc.CancelImageTask(taskID); err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Message(c, "task cancelled")
}

// PreviewPayload handles POST /studio/image/preview
func (h *Handler) PreviewPayload(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.PreviewImagePayload(&req)
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
