package studio

import (
	"encoding/json"
	"net/http"
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

// ─── Keys ─────────────────────────────────────────────────────

func (h *Handler) ListKeys(c *gin.Context) {
	result, err := h.svc.ListKeys()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) AddKey(c *gin.Context) {
	var req AddKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}
	if req.Value == "" {
		utils.BadRequest(c, "Missing API key value")
		return
	}
	result, err := h.svc.AddKey(req)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) ActivateKey(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.ActivateKey(id)
	if err != nil {
		if err.Error() == "key not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) DeleteKey(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.DeleteKey(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) UpdateKey(c *gin.Context) {
	id := c.Param("id")
	var req UpdateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}
	result, err := h.svc.UpdateKey(id, req)
	if err != nil {
		if err.Error() == "key not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

// ─── Presets ──────────────────────────────────────────────────

func (h *Handler) GetPresets(c *gin.Context) {
	data, err := h.svc.GetPresets()
	if err != nil {
		utils.InternalError(c, "Failed to load presets")
		return
	}
	var presets interface{}
	if err := json.Unmarshal(data, &presets); err != nil {
		utils.InternalError(c, "Invalid presets file")
		return
	}
	utils.Success(c, presets)
}

// ─── Compile Prompt ───────────────────────────────────────────

func (h *Handler) CompilePrompt(c *gin.Context) {
	var sel Selection
	if err := c.ShouldBindJSON(&sel); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}
	text := h.svc.CompilePrompt(sel)
	utils.Success(c, gin.H{"prompt": text})
}

// ─── Generate Seedance ────────────────────────────────────────

func (h *Handler) Generate(c *gin.Context) {
	var sel Selection
	if err := c.ShouldBindJSON(&sel); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}
	result, err := h.svc.Generate(sel)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

// ─── Status ───────────────────────────────────────────────────

func (h *Handler) GetStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	result, err := h.svc.GetStatus(taskID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

// ─── Cancel Task ──────────────────────────────────────────────

func (h *Handler) CancelTask(c *gin.Context) {
	taskID := c.Param("taskId")
	result, err := h.svc.CancelTask(taskID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

// ─── Seedream ─────────────────────────────────────────────────

func (h *Handler) ListTrustedAssets(c *gin.Context) {
	utils.Success(c, h.svc.ListTrustedAssets())
}

func (h *Handler) GenerateSeedream(c *gin.Context) {
	var req SeedreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}
	if req.Prompt == "" {
		utils.BadRequest(c, "Prompt is required.")
		return
	}
	result, err := h.svc.GenerateSeedream(req)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

// ─── Assets API ───────────────────────────────────────────────

func (h *Handler) CreateAssetGroup(c *gin.Context) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ProjectName string `json:"projectName"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}
	result, err := h.svc.CreateAssetGroup(body.Name, body.Description, body.ProjectName)
	if err != nil {
		utils.Error(c, errorStatus(err), err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) ListAssetGroups(c *gin.Context) {
	result, err := h.svc.ListAssetGroups()
	if err != nil {
		utils.Error(c, errorStatus(err), err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) CreateAsset(c *gin.Context) {
	var body struct {
		GroupID            string `json:"groupId"`
		URL                string `json:"url"`
		Name               string `json:"name"`
		AssetType          string `json:"assetType"`
		ModerationStrategy string `json:"moderationStrategy"`
		ProjectName        string `json:"projectName"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}
	result, err := h.svc.CreateAsset(body.GroupID, body.URL, body.Name, body.AssetType, body.ModerationStrategy, body.ProjectName)
	if err != nil {
		utils.Error(c, errorStatus(err), err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) GetAsset(c *gin.Context) {
	assetID := c.Param("id")
	projectName := c.Query("projectName")
	result, err := h.svc.GetAsset(assetID, projectName)
	if err != nil {
		utils.Error(c, errorStatus(err), err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) ListAssets(c *gin.Context) {
	groupID := c.Query("groupId")
	statuses := c.Query("statuses")
	projectName := c.Query("projectName")
	result, err := h.svc.ListAssets(groupID, statuses, projectName)
	if err != nil {
		utils.Error(c, errorStatus(err), err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) DeleteAsset(c *gin.Context) {
	assetID := c.Param("id")
	projectName := c.Query("projectName")
	result, err := h.svc.DeleteAsset(assetID, projectName)
	if err != nil {
		utils.Error(c, errorStatus(err), err.Error())
		return
	}
	utils.Success(c, result)
}

// ─── Health & Debug ───────────────────────────────────────────

func (h *Handler) Health(c *gin.Context) {
	utils.Success(c, h.svc.Health())
}

func (h *Handler) Debug(c *gin.Context) {
	utils.Success(c, h.svc.Debug())
}

// ─── Helpers ──────────────────────────────────────────────────

func errorStatus(err error) int {
	msg := err.Error()
	if strings.Contains(msg, "required") {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
