package preset

import (
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

// ─── Groups ─────────────────────────────────────────────────────

func (h *Handler) ListGroups(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"
	groups, err := h.svc.ListGroups(includeInactive)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, groups)
}

func (h *Handler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.CreateGroup(&req)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "group slug already exists")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, result)
}

func (h *Handler) UpdateGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}
	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.UpdateGroup(id, &req)
	if err != nil {
		if err.Error() == "group not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

// ─── Presets ────────────────────────────────────────────────────

func (h *Handler) ListPresets(c *gin.Context) {
	groupID := c.Query("group_id")
	includeInactive := c.Query("include_inactive") == "true"
	presets, err := h.svc.ListPresets(groupID, includeInactive)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, presets)
}

func (h *Handler) GetPreset(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}
	result, err := h.svc.GetPresetByID(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if result == nil {
		utils.NotFound(c, "preset not found")
		return
	}
	utils.Success(c, result)
}

func (h *Handler) CreatePreset(c *gin.Context) {
	var req CreatePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.CreatePreset(&req)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "preset code already exists in this group")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, result)
}

func (h *Handler) UpdatePreset(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}
	var req UpdatePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.UpdatePreset(id, &req)
	if err != nil {
		if err.Error() == "preset not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) SoftDeletePreset(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}
	if err := h.svc.SoftDeletePreset(id); err != nil {
		if err.Error() == "preset not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "preset deleted")
}
