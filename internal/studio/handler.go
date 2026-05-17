package studio

import (
	"errors"
	"strings"

	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ─── Legacy endpoint (Selection-based) ─────────────────────────

func (h *Handler) Generate(c *gin.Context) {
	var req Selection
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.Generate(&req)
	if err != nil {
		if err.Error() == "model not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Created(c, result)
}

// ─── New unified endpoint ───────────────────────────────────────

// GenerateUnified handles POST /studio/generate with the unified payload.
//
//	{
//	  "model": "dreamina-seedance-2-0-fast-260128",
//	  "content": [
//	    {"type": "text", "text": "prompt here"},
//	    {"type": "image", "id": "uuid", "name": "img.png", "text": "description"}
//	  ],
//	  "ratio": "16:9",
//	  "duration": 5,
//	  "camerafixed": false,
//	  "seed": "22",
//	  "quality": "standard",
//	  "quantity": 1,
//	  "watermark": false,
//	  "resolution": "480p",
//	  "generate_audio": true,
//	  "image_mode": "PIL"
//	}
func (h *Handler) GenerateUnified(c *gin.Context) {
	var req StudioGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.GenerateUnified(&req)
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

// ─── Status and cancellation ─────────────────────────────────────

func (h *Handler) GetStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "taskId is required")
		return
	}

	// Try unified first, fall back to legacy
	result, err := h.svc.GetStatusUnified(taskID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

// GetStatusLegacy handles the legacy status response format.
func (h *Handler) GetStatusLegacy(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "taskId is required")
		return
	}

	result, err := h.svc.GetStatus(taskID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

// ─── Asset sync ─────────────────────────────────────────────────

func (h *Handler) SyncAsset(c *gin.Context) {
	var req SyncAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.SyncAsset(&req)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) ListSyncedAssets(c *gin.Context) {
	modelID := c.Query("model_id")
	if modelID == "" {
		utils.BadRequest(c, "model_id query parameter is required")
		return
	}

	assets, err := h.svc.ListSyncedAssets(modelID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, assets)
}

// ─── Enriched file listing ──────────────────────────────────────

func (h *Handler) ListFilesWithSync(c *gin.Context) {
	category := c.Query("category")
	storage := c.Query("storage")
	trashed := c.Query("trashed") == "true"

	files, err := h.svc.GetFilesWithSync(category, storage, trashed)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, files)
}

func (h *Handler) ListCharacterFilesWithSync(c *gin.Context) {
	characterID := c.Param("id")
	if characterID == "" {
		utils.BadRequest(c, "character id is required")
		return
	}

	files, err := h.svc.GetCharacterFilesWithSync(characterID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, files)
}

func (h *Handler) SyncCharacterAssets(c *gin.Context) {
	var req SyncCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.SyncCharacterAssets(&req)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) CancelTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "taskId is required")
		return
	}

	if err := h.svc.CancelTask(taskID); err != nil {
		if errors.Is(err, errors.New("unknown task")) {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Message(c, "task cancelled")
}

// ─── Generation log CRUD ──────────────────────────────────────────

// ListGenerationLogs returns paginated generation logs.
func (h *Handler) ListGenerationLogs(c *gin.Context) {
	var req ListGenerationLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	result, err := h.svc.ListGenerationLogs(req.Page, req.Limit, req.ProjectID, req.SceneID, req.Status, req.ModelName, req.UserID, req.DateFrom, req.DateTo)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

// GetGenerationLog returns a single generation log by its ID.
func (h *Handler) GetGenerationLog(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}

	log, err := h.svc.GetGenerationLog(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, log)
}

// PreviewPayload returns the AI API payload without sending it or saving logs.
func (h *Handler) PreviewPayload(c *gin.Context) {
	var req StudioGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.PreviewPayload(&req)
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
