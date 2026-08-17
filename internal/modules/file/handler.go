package file

import (
	"strconv"

	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Upload(c *gin.Context) {
	category := c.PostForm("category")
	if category == "" {
		utils.BadRequest(c, "category is required")
		return
	}

	storage := c.DefaultPostForm("storage", "persistent")

	file, err := c.FormFile("file")
	if err != nil {
		utils.BadRequest(c, "file field is required")
		return
	}

	f, err := file.Open()
	if err != nil {
		utils.InternalError(c, "failed to read file")
		return
	}
	defer f.Close()

	data := make([]byte, file.Size)
	if _, err := f.Read(data); err != nil {
		utils.InternalError(c, "failed to read file data")
		return
	}

	result, err := h.svc.Upload(data, file.Filename, category, storage)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) GetFile(c *gin.Context) {
	id := c.Param("id")
	f, err := h.svc.GetFile(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if f == nil {
		utils.NotFound(c, "file not found")
		return
	}
	utils.Success(c, f)
}

func (h *Handler) ServeFile(c *gin.Context) {
	id := c.Param("id")
	path, err := h.svc.GetServePath(id)
	if err != nil {
		if err.Error() == "file not found" {
			utils.NotFound(c, err.Error())
			return
		}
		if err.Error() == "file has been deleted" {
			utils.Gone(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	c.File(path)
}

func (h *Handler) ServeThumbnail(c *gin.Context) {
	id := c.Param("id")
	path, err := h.svc.GetThumbnailPath(id)
	if err != nil {
		if err.Error() == "file not found" {
			utils.NotFound(c, err.Error())
			return
		}
		if err.Error() == "file has been deleted" {
			utils.Gone(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	c.File(path)
}

// ServeVision serves the vision-size rendering of an image (the sweet spot
// between the 300px thumbnail and the full original) for Claude vision
// analysis. Public so Anthropic's servers can fetch it without auth.
func (h *Handler) ServeVision(c *gin.Context) {
	id := c.Param("id")
	path, err := h.svc.GetVisionPath(id)
	if err != nil {
		switch err.Error() {
		case "file not found":
			utils.NotFound(c, err.Error())
		case "file has been deleted":
			utils.Gone(c, err.Error())
		case "file is not an image":
			utils.NotFound(c, err.Error())
		default:
			utils.InternalError(c, err.Error())
		}
		return
	}
	c.File(path)
}

func (h *Handler) SoftDelete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.SoftDelete(id); err != nil {
		if err.Error() == "file not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.BadRequest(c, err.Error())
		return
	}
	utils.Message(c, "file moved to trash")
}

func (h *Handler) Restore(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Restore(id); err != nil {
		if err.Error() == "file not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.BadRequest(c, err.Error())
		return
	}
	utils.Message(c, "file restored")
}

func (h *Handler) RecoverTemp(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.RecoverTemp(id); err != nil {
		if err.Error() == "file not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.BadRequest(c, err.Error())
		return
	}
	utils.Message(c, "temp file recovered")
}

func (h *Handler) HardDelete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.HardDelete(id); err != nil {
		if err.Error() == "file not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "file permanently deleted")
}

func (h *Handler) ListFilesPage(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	category := c.Query("category")
	storage := c.Query("storage")
	search := c.Query("q")

	result, err := h.svc.ListFilesPage(page, pageSize, category, storage, search)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, result)
}

func (h *Handler) ListFiles(c *gin.Context) {
	category := c.Query("category")
	storage := c.Query("storage")
	trashed := c.Query("trashed") == "true"

	files, err := h.svc.ListFiles(category, storage, trashed)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if files == nil {
		files = []File{}
	}
	utils.Success(c, files)
}

func (h *Handler) ListTrash(c *gin.Context) {
	files, err := h.svc.ListTrash()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if files == nil {
		files = []File{}
	}
	utils.Success(c, files)
}
