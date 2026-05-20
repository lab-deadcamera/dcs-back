package image

import (
	"os"
	"strings"

	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

type uploadBase64Request struct {
	Filename string `json:"filename" binding:"required"`
	Data     string `json:"data" binding:"required"`
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Upload(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		h.uploadMultipart(c)
		return
	}

	h.uploadBase64(c)
}

func (h *Handler) uploadMultipart(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		utils.BadRequest(c, "image field is required")
		return
	}

	result, err := h.svc.Upload(file)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) uploadBase64(c *gin.Context) {
	var req uploadBase64Request
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "JSON body with filename and data fields is required")
		return
	}

	result, err := h.svc.UploadBase64(req.Filename, req.Data)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Created(c, result)
}

func (h *Handler) Serve(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		utils.BadRequest(c, "filename is required")
		return
	}

	path := h.svc.GetPath(filename)
	if !fileExists(path) {
		utils.NotFound(c, "image not found")
		return
	}

	c.File(path)
}

func (h *Handler) ServeThumbnail(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		utils.BadRequest(c, "filename is required")
		return
	}

	path := h.svc.GetThumbnailPath(filename)
	if !fileExists(path) {
		utils.NotFound(c, "thumbnail not found")
		return
	}

	c.File(path)
}

func (h *Handler) Delete(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		utils.BadRequest(c, "filename is required")
		return
	}

	if err := h.svc.Delete(filename); err != nil {
		utils.InternalError(c, "failed to delete image")
		return
	}

	utils.Message(c, "image deleted")
}

func (h *Handler) List(c *gin.Context) {
	files, err := h.svc.List()
	if err != nil {
		utils.InternalError(c, "failed to list images")
		return
	}

	var result []map[string]string
	for _, f := range files {
		result = append(result, map[string]string{
			"filename":      f,
			"url":           h.svc.baseURL + "/api/v1/images/" + f,
			"thumbnail_url": h.svc.baseURL + "/api/v1/images/thumbnails/" + f,
		})
	}

	utils.Success(c, result)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
