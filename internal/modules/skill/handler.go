package skill

import (
	"fmt"
	"net/http"

	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, fmt.Sprintf("invalid request: %v", err))
		return
	}

	skill, err := h.svc.Create(&req)
	if err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to create skill: %v", err))
		return
	}

	utils.Success(c, skill)
}

func (h *Handler) List(c *gin.Context) {
	skills, err := h.svc.List()
	if err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to list skills: %v", err))
		return
	}

	if skills == nil {
		skills = []Skill{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    skills,
	})
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "skill id is required")
		return
	}

	skill, err := h.svc.GetByID(id)
	if err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to get skill: %v", err))
		return
	}
	if skill == nil {
		utils.NotFound(c, "skill not found")
		return
	}

	utils.Success(c, skill)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "skill id is required")
		return
	}

	var req UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if err := h.svc.Update(id, &req); err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to update skill: %v", err))
		return
	}

	// Return the updated skill
	skill, err := h.svc.GetByID(id)
	if err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to get updated skill: %v", err))
		return
	}

	utils.Success(c, skill)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "skill id is required")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to delete skill: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "skill deleted",
	})
}
