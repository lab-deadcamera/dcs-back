package project

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

// ─── Projects ───────────────────────────────────────────────────

func (h *Handler) Create(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.Create(&req)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) List(c *gin.Context) {
	projects, err := h.svc.List()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, projects)
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}

	result, err := h.svc.GetProjectWithScenes(id)
	if err != nil {
		if err.Error() == "project not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.Update(id, &req)
	if err != nil {
		if err.Error() == "project not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) SoftDelete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}

	if err := h.svc.SoftDelete(id); err != nil {
		if err.Error() == "project not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Message(c, "project deleted")
}

// ─── Scenes ─────────────────────────────────────────────────────

func (h *Handler) CreateScene(c *gin.Context) {
	projectID := c.Param("id")

	var req CreateSceneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.CreateScene(projectID, &req)
	if err != nil {
		if err.Error() == "project not found" {
			utils.NotFound(c, err.Error())
			return
		}
		// Check for unique constraint violation (duplicate scene number)
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "scene number already exists for this project")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) ListScenes(c *gin.Context) {
	projectID := c.Param("id")

	scenes, err := h.svc.ListScenes(projectID)
	if err != nil {
		if err.Error() == "project not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, scenes)
}

func (h *Handler) GetSceneByID(c *gin.Context) {
	sceneID := c.Param("sceneId")
	if sceneID == "" {
		utils.BadRequest(c, "sceneId is required")
		return
	}

	result, err := h.svc.GetSceneWithTakes(sceneID)
	if err != nil {
		if err.Error() == "scene not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) UpdateScene(c *gin.Context) {
	sceneID := c.Param("sceneId")
	if sceneID == "" {
		utils.BadRequest(c, "sceneId is required")
		return
	}

	var req UpdateSceneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.UpdateScene(sceneID, &req)
	if err != nil {
		if err.Error() == "scene not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) SoftDeleteScene(c *gin.Context) {
	sceneID := c.Param("sceneId")
	if sceneID == "" {
		utils.BadRequest(c, "sceneId is required")
		return
	}

	if err := h.svc.SoftDeleteScene(sceneID); err != nil {
		if err.Error() == "scene not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Message(c, "scene deleted")
}

// ─── Takes ──────────────────────────────────────────────────────

func (h *Handler) CreateTake(c *gin.Context) {
	sceneID := c.Param("sceneId")

	var req CreateTakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.CreateTake(sceneID, &req)
	if err != nil {
		if err.Error() == "scene not found" {
			utils.NotFound(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "take number already exists for this scene")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) ListTakes(c *gin.Context) {
	sceneID := c.Param("sceneId")

	takes, err := h.svc.ListTakes(sceneID)
	if err != nil {
		if err.Error() == "scene not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, takes)
}

func (h *Handler) GetTakeByID(c *gin.Context) {
	takeID := c.Param("takeId")
	if takeID == "" {
		utils.BadRequest(c, "takeId is required")
		return
	}

	result, err := h.svc.GetTakeByID(takeID)
	if err != nil {
		if err.Error() == "take not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) UpdateTake(c *gin.Context) {
	takeID := c.Param("takeId")
	if takeID == "" {
		utils.BadRequest(c, "takeId is required")
		return
	}

	var req UpdateTakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.UpdateTake(takeID, &req)
	if err != nil {
		if err.Error() == "take not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) SoftDeleteTake(c *gin.Context) {
	takeID := c.Param("takeId")
	if takeID == "" {
		utils.BadRequest(c, "takeId is required")
		return
	}

	if err := h.svc.SoftDeleteTake(takeID); err != nil {
		if err.Error() == "take not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Message(c, "take deleted")
}
