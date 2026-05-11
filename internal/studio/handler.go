package studio

import (
	"encoding/json"
	"net/http"
	"strings"

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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) AddKey(c *gin.Context) {
	var req AddKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	if req.Value == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Missing API key value"})
		return
	}
	result, err := h.svc.AddKey(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ActivateKey(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.ActivateKey(id)
	if err != nil {
		if err.Error() == "key not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) DeleteKey(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.DeleteKey(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) UpdateKey(c *gin.Context) {
	id := c.Param("id")
	var req UpdateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	result, err := h.svc.UpdateKey(id, req)
	if err != nil {
		if err.Error() == "key not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ─── Presets ──────────────────────────────────────────────────

func (h *Handler) GetPresets(c *gin.Context) {
	data, err := h.svc.GetPresets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to load presets"})
		return
	}
	var presets interface{}
	if err := json.Unmarshal(data, &presets); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Invalid presets file"})
		return
	}
	c.JSON(http.StatusOK, presets)
}

// ─── Compile Prompt ───────────────────────────────────────────

func (h *Handler) CompilePrompt(c *gin.Context) {
	var sel Selection
	if err := c.ShouldBindJSON(&sel); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	text := h.svc.CompilePrompt(sel)
	c.JSON(http.StatusOK, gin.H{"prompt": text})
}

// ─── Generate Seedance ────────────────────────────────────────

func (h *Handler) Generate(c *gin.Context) {
	var sel Selection
	if err := c.ShouldBindJSON(&sel); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	result, err := h.svc.Generate(sel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ─── Status ───────────────────────────────────────────────────

func (h *Handler) GetStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	result, err := h.svc.GetStatus(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ─── Cancel Task ──────────────────────────────────────────────

func (h *Handler) CancelTask(c *gin.Context) {
	taskID := c.Param("taskId")
	result, err := h.svc.CancelTask(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ─── Seedream ─────────────────────────────────────────────────

func (h *Handler) ListTrustedAssets(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.ListTrustedAssets())
}

func (h *Handler) GenerateSeedream(c *gin.Context) {
	var req SeedreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Prompt is required."})
		return
	}
	result, err := h.svc.GenerateSeedream(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ─── Assets API ───────────────────────────────────────────────

func (h *Handler) CreateAssetGroup(c *gin.Context) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ProjectName string `json:"projectName"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	result, err := h.svc.CreateAssetGroup(body.Name, body.Description, body.ProjectName)
	if err != nil {
		c.JSON(errorStatus(err), ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListAssetGroups(c *gin.Context) {
	result, err := h.svc.ListAssetGroups()
	if err != nil {
		c.JSON(errorStatus(err), ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	result, err := h.svc.CreateAsset(body.GroupID, body.URL, body.Name, body.AssetType, body.ModerationStrategy, body.ProjectName)
	if err != nil {
		c.JSON(errorStatus(err), ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetAsset(c *gin.Context) {
	assetID := c.Param("id")
	projectName := c.Query("projectName")
	result, err := h.svc.GetAsset(assetID, projectName)
	if err != nil {
		c.JSON(errorStatus(err), ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListAssets(c *gin.Context) {
	groupID := c.Query("groupId")
	statuses := c.Query("statuses")
	projectName := c.Query("projectName")
	result, err := h.svc.ListAssets(groupID, statuses, projectName)
	if err != nil {
		c.JSON(errorStatus(err), ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) DeleteAsset(c *gin.Context) {
	assetID := c.Param("id")
	projectName := c.Query("projectName")
	result, err := h.svc.DeleteAsset(assetID, projectName)
	if err != nil {
		c.JSON(errorStatus(err), ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ─── Health & Debug ───────────────────────────────────────────

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.Health())
}

func (h *Handler) Debug(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.Debug())
}

// ─── Helpers ──────────────────────────────────────────────────

func errorStatus(err error) int {
	// Simple heuristic: if the error contains common client error patterns
	msg := err.Error()
	if strings.Contains(msg, "required") {
		return http.StatusBadRequest
	}
	// Default to 500 for upstream API errors
	return http.StatusInternalServerError
}
