package character

import (
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
	var req CreateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	ch, err := h.svc.Create(req)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, ch)
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	ch, err := h.svc.GetByIDWithFiles(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if ch == nil {
		utils.NotFound(c, "character not found")
		return
	}
	utils.Success(c, ch)
}

func (h *Handler) List(c *gin.Context) {
	chars, err := h.svc.List()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if chars == nil {
		chars = []Character{}
	}
	utils.Success(c, chars)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	ch, err := h.svc.Update(id, req)
	if err != nil {
		if err.Error() == "character not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, ch)
}

func (h *Handler) SoftDelete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.SoftDelete(id); err != nil {
		if err.Error() == "character not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "character deleted")
}

func (h *Handler) AddFile(c *gin.Context) {
	characterID := c.Param("id")
	var req AddFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.AddFile(characterID, req.FileID, req.Role); err != nil {
		if err.Error() == "character not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "file added to character")
}

func (h *Handler) RemoveFile(c *gin.Context) {
	characterID := c.Param("id")
	fileID := c.Param("fileId")
	if err := h.svc.RemoveFile(characterID, fileID); err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "file removed from character")
}

func (h *Handler) ListFiles(c *gin.Context) {
	characterID := c.Param("id")
	files, err := h.svc.ListFiles(characterID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if files == nil {
		files = []FileRef{}
	}
	utils.Success(c, files)
}
