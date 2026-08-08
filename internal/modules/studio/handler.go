package studio

import (
	"fmt"
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

// ─── Gallery (external galleries admin view) ───────────────────

func (h *Handler) ListGalleryModels(c *gin.Context) {
	models, err := h.svc.ListGalleryModels()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, models)
}

func (h *Handler) ListGalleryModelAssets(c *gin.Context) {
	modelID := c.Param("modelId")
	if modelID == "" {
		utils.BadRequest(c, "modelId is required")
		return
	}

	assets, err := h.svc.ListGalleryAssets(modelID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, assets)
}

func (h *Handler) ListGalleryErrors(c *gin.Context) {
	modelID := c.Query("model_id")
	fileID := c.Query("file_id")
	if modelID == "" || fileID == "" {
		utils.BadRequest(c, "model_id and file_id query parameters are required")
		return
	}

	errors, err := h.svc.ListGalleryErrors(modelID, fileID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, errors)
}

func (h *Handler) FixAsset(c *gin.Context) {
	var req FixAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.FixAsset(&req)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

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

	result, err := h.svc.ListGenerationLogs(req.Page, req.Limit, req.ProjectID, req.SceneID, req.Status, req.ModelName, req.UserID, req.DateFrom, req.DateTo, req.ResourceType)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

// GetGenerationLogsCostSummary returns the total cost for filtered generation logs.
func (h *Handler) GetGenerationLogsCostSummary(c *gin.Context) {
	var req ListGenerationLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.SumGenerationLogsCost(req.ProjectID, req.SceneID, req.Status, req.ModelName, req.UserID, req.DateFrom, req.DateTo, req.ResourceType)
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

// ListServerCommunications returns paginated server-to-server communication logs.
func (h *Handler) ListServerCommunications(c *gin.Context) {
	taskID := c.Query("task_id")
	modelName := c.Query("model_name")
	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	result, err := h.svc.ListServerCommunications(taskID, modelName, page, limit)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

// GetServerCommunication returns a single server communication log by ID.
func (h *Handler) GetServerCommunication(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}

	log, err := h.svc.GetServerCommunication(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if log == nil {
		utils.NotFound(c, "server communication not found")
		return
	}

	utils.Success(c, log)
}
