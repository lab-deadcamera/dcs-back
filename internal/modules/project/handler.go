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

func (h *Handler) ListAll(c *gin.Context) {
	projects, err := h.svc.ListAll()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, projects)
}

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

	result, err := h.svc.GetProjectWithChapters(id)
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

// ─── Chapters ───────────────────────────────────────────────────

func (h *Handler) CreateChapter(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		utils.BadRequest(c, "id is required")
		return
	}

	var req CreateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.CreateChapter(projectID, &req)
	if err != nil {
		if err.Error() == "project not found" {
			utils.NotFound(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "chapter number already exists for this project")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) ListChapters(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		utils.BadRequest(c, "id is required")
		return
	}

	chapters, err := h.svc.ListChapters(projectID)
	if err != nil {
		if err.Error() == "project not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, chapters)
}

func (h *Handler) GetChapterByID(c *gin.Context) {
	chapterID := c.Param("chapterId")
	if chapterID == "" {
		utils.BadRequest(c, "chapterId is required")
		return
	}

	chapter, err := h.svc.GetChapterByID(chapterID)
	if err != nil {
		if err.Error() == "chapter not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, chapter)
}

func (h *Handler) UpdateChapter(c *gin.Context) {
	chapterID := c.Param("chapterId")
	if chapterID == "" {
		utils.BadRequest(c, "chapterId is required")
		return
	}

	var req UpdateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.UpdateChapter(chapterID, &req)
	if err != nil {
		if err.Error() == "chapter not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) SoftDeleteChapter(c *gin.Context) {
	chapterID := c.Param("chapterId")
	if chapterID == "" {
		utils.BadRequest(c, "chapterId is required")
		return
	}

	if err := h.svc.SoftDeleteChapter(chapterID); err != nil {
		if err.Error() == "chapter not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Message(c, "chapter deleted")
}

// ─── Scenes ─────────────────────────────────────────────────────

func (h *Handler) CreateScene(c *gin.Context) {
	chapterID := c.Param("chapterId")
	if chapterID == "" {
		utils.BadRequest(c, "chapterId is required")
		return
	}

	var req CreateSceneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.CreateScene(chapterID, &req)
	if err != nil {
		if err.Error() == "chapter not found" {
			utils.NotFound(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "scene number already exists for this chapter")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) ListScenes(c *gin.Context) {
	chapterID := c.Param("chapterId")
	if chapterID == "" {
		utils.BadRequest(c, "chapterId is required")
		return
	}

	scenes, err := h.svc.ListScenes(chapterID)
	if err != nil {
		if err.Error() == "chapter not found" {
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

	result, err := h.svc.GetSceneWithShots(sceneID)
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

// ─── Shots ──────────────────────────────────────────────────────

func (h *Handler) CreateShot(c *gin.Context) {
	sceneID := c.Param("sceneId")
	if sceneID == "" {
		utils.BadRequest(c, "sceneId is required")
		return
	}

	var req CreateShotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.CreateShot(sceneID, &req)
	if err != nil {
		if err.Error() == "scene not found" {
			utils.NotFound(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "shot number already exists for this scene")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) ListShots(c *gin.Context) {
	sceneID := c.Param("sceneId")
	if sceneID == "" {
		utils.BadRequest(c, "sceneId is required")
		return
	}

	shots, err := h.svc.ListShots(sceneID)
	if err != nil {
		if err.Error() == "scene not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, shots)
}

func (h *Handler) GetShotByID(c *gin.Context) {
	shotID := c.Param("shotId")
	if shotID == "" {
		utils.BadRequest(c, "shotId is required")
		return
	}

	result, err := h.svc.GetShotWithTakes(shotID)
	if err != nil {
		if err.Error() == "shot not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) UpdateShot(c *gin.Context) {
	shotID := c.Param("shotId")
	if shotID == "" {
		utils.BadRequest(c, "shotId is required")
		return
	}

	var req UpdateShotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.UpdateShot(shotID, &req)
	if err != nil {
		if err.Error() == "shot not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

func (h *Handler) SoftDeleteShot(c *gin.Context) {
	shotID := c.Param("shotId")
	if shotID == "" {
		utils.BadRequest(c, "shotId is required")
		return
	}

	if err := h.svc.SoftDeleteShot(shotID); err != nil {
		if err.Error() == "shot not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Message(c, "shot deleted")
}

// ─── Takes ──────────────────────────────────────────────────────

func (h *Handler) CreateTake(c *gin.Context) {
	shotID := c.Param("shotId")
	if shotID == "" {
		utils.BadRequest(c, "shotId is required")
		return
	}

	var req CreateTakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.CreateTake(shotID, &req)
	if err != nil {
		if err.Error() == "shot not found" {
			utils.NotFound(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "take number already exists for this shot")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) ListTakes(c *gin.Context) {
	shotID := c.Param("shotId")
	if shotID == "" {
		utils.BadRequest(c, "shotId is required")
		return
	}

	takes, err := h.svc.ListTakes(shotID)
	if err != nil {
		if err.Error() == "shot not found" {
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

func (h *Handler) SaveGeneration(c *gin.Context) {
	shotID := c.Param("shotId")
	if shotID == "" {
		utils.BadRequest(c, "shotId is required")
		return
	}

	var req SaveGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.SaveGeneration(shotID, &req)
	if err != nil {
		if err.Error() == "shot not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) ToggleTakeActive(c *gin.Context) {
	takeID := c.Param("takeId")

	result, err := h.svc.ToggleTakeActive(takeID)
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

// DownloadTake triggers a local download of the take's external video.
func (h *Handler) DownloadTake(c *gin.Context) {
	takeID := c.Param("takeId")
	if takeID == "" {
		utils.BadRequest(c, "takeId is required")
		return
	}

	username, _ := c.Get("username")
	userStr, _ := username.(string)

	result, err := h.svc.DownloadTakeVideo(takeID, userStr)
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

// ─── Scene Assignments ──────────────────────────────────────────

func (h *Handler) GetSceneAssignments(c *gin.Context) {
	sceneID := c.Param("sceneId")
	if sceneID == "" {
		utils.BadRequest(c, "sceneId is required")
		return
	}
	result, err := h.svc.GetSceneAssignments(sceneID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) AssignPresetToScene(c *gin.Context) {
	sceneID := c.Param("sceneId")
	var req struct {
		PresetID string `json:"preset_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	id, err := h.svc.AssignPresetToScene(sceneID, req.PresetID)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "preset already assigned to this scene")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, map[string]string{"id": id})
}

func (h *Handler) AssignCharacterToScene(c *gin.Context) {
	sceneID := c.Param("sceneId")
	var req struct {
		CharacterID string `json:"character_id" binding:"required"`
		Slot        string `json:"slot"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	id, err := h.svc.AssignCharacterToScene(sceneID, req.CharacterID, req.Slot)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "character already assigned to this scene")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, map[string]string{"id": id})
}

func (h *Handler) AssignAssetToScene(c *gin.Context) {
	sceneID := c.Param("sceneId")
	var req struct {
		FileID string `json:"file_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	id, err := h.svc.AssignAssetToScene(sceneID, req.FileID)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "asset already assigned to this scene")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, map[string]string{"id": id})
}

func (h *Handler) RemoveScenePreset(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	if err := h.svc.RemoveScenePreset(assignmentID); err != nil {
		if err.Error() == "assignment not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "preset unassigned")
}

func (h *Handler) RemoveSceneCharacter(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	if err := h.svc.RemoveSceneCharacter(assignmentID); err != nil {
		if err.Error() == "assignment not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "character unassigned")
}

func (h *Handler) RemoveSceneAsset(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	if err := h.svc.RemoveSceneAsset(assignmentID); err != nil {
		if err.Error() == "assignment not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "asset unassigned")
}

// ─── Shot Resources ────────────────────────────────────────────────


func (h *Handler) AssignCharacterToShot(c *gin.Context) {
	shotID := c.Param("shotId")
	var req struct {
		CharacterID string `json:"character_id" binding:"required"`
		Slot        string `json:"slot"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	id, err := h.svc.AssignCharacterToShot(shotID, req.CharacterID, req.Slot)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "character already assigned to this shot")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, map[string]string{"id": id})
}

func (h *Handler) AssignAssetToShot(c *gin.Context) {
	shotID := c.Param("shotId")
	var req struct {
		FileID string `json:"file_id" binding:"required"`
		Slot   string `json:"slot"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	if req.Slot == "" {
		req.Slot = "free"
	}
	id, err := h.svc.AssignAssetToShot(shotID, req.FileID, req.Slot)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "asset already assigned to this shot")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, map[string]string{"id": id})
}

func (h *Handler) AssignPresetToShot(c *gin.Context) {
	shotID := c.Param("shotId")
	var req struct {
		PresetID string `json:"preset_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	id, err := h.svc.AssignPresetToShot(shotID, req.PresetID)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			utils.BadRequest(c, "preset already assigned to this shot")
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, map[string]string{"id": id})
}

func (h *Handler) RemoveShotCharacter(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	if err := h.svc.RemoveShotCharacter(assignmentID); err != nil {
		if err.Error() == "assignment not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "character unassigned")
}

func (h *Handler) RemoveShotAsset(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	if err := h.svc.RemoveShotAsset(assignmentID); err != nil {
		if err.Error() == "assignment not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "asset unassigned")
}

func (h *Handler) RemoveShotPreset(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	if err := h.svc.RemoveShotPreset(assignmentID); err != nil {
		if err.Error() == "assignment not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "preset unassigned")
}

func (h *Handler) UpdateShotModel(c *gin.Context) {
	shotID := c.Param("shotId")
	var req struct {
		ModelID string `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.UpdateShotModel(shotID, req.ModelID); err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "model updated")
}
