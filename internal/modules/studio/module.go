package studio

import (
	"github.com/gin-gonic/gin"
)

type Module struct {
	hdl      *Handler
	videoHdl videoHandler
	imageHdl imageHandler
	audioHdl audioHandler
	textHdl  textHandler
}

type videoHandler interface {
	Generate(c *gin.Context)
	GetStatus(c *gin.Context)
	CancelTask(c *gin.Context)
	PreviewPayload(c *gin.Context)
}

type imageHandler interface {
	Generate(c *gin.Context)
	GetStatus(c *gin.Context)
	CancelTask(c *gin.Context)
	PreviewPayload(c *gin.Context)
}

type audioHandler interface {
	Generate(c *gin.Context)
	GetStatus(c *gin.Context)
	CancelTask(c *gin.Context)
	PreviewPayload(c *gin.Context)
}

type textHandler interface {
	Generate(c *gin.Context)
	GetStatus(c *gin.Context)
	CancelTask(c *gin.Context)
	PreviewPayload(c *gin.Context)
	ClaudeGenerateShots(c *gin.Context)
	ClaudeGenerateShotsV2(c *gin.Context)
	GetClaudeShotsStatus(c *gin.Context)
	ClaudeRefineShots(c *gin.Context)
	ClaudeOptimizePrompt(c *gin.Context)
	ListGenerateShotsLogs(c *gin.Context)
	GetGenerateShotsLog(c *gin.Context)
}

func NewModule(hdl *Handler, videoHdl videoHandler, imageHdl imageHandler, audioHdl audioHandler, textHdl textHandler) *Module {
	return &Module{
		hdl:      hdl,
		videoHdl: videoHdl,
		imageHdl: imageHdl,
		audioHdl: audioHdl,
		textHdl:  textHdl,
	}
}

func (m *Module) Name() string { return "studio" }

func (m *Module) Register(rg *gin.RouterGroup, authMw, adminMw gin.HandlerFunc) {
	g := rg.Group("/studio")
	g.Use(authMw)
	{
		// Asset sync
		g.POST("/sync-asset", m.hdl.SyncAsset)
		g.GET("/synced-assets", m.hdl.ListSyncedAssets)
		g.GET("/files-with-sync", m.hdl.ListFilesWithSync)
		g.GET("/characters/:id/files-with-sync", m.hdl.ListCharacterFilesWithSync)
		g.POST("/sync-character-assets", m.hdl.SyncCharacterAssets)

		// External gallery admin view
		gallery := g.Group("/gallery")
		gallery.Use(adminMw)
		{
			gallery.GET("/models", m.hdl.ListGalleryModels)
			gallery.GET("/models/:modelId/assets", m.hdl.ListGalleryModelAssets)
			gallery.GET("/errors", m.hdl.ListGalleryErrors)
			gallery.POST("/fix-asset", m.hdl.FixAsset)
		}

		// Logs
		g.GET("/logs/generation", m.hdl.ListGenerationLogs)
		g.GET("/logs/generation/cost-summary", m.hdl.GetGenerationLogsCostSummary)
		g.GET("/logs/generation/:id", m.hdl.GetGenerationLog)
		g.GET("/logs/server-communications", m.hdl.ListServerCommunications)
		g.GET("/logs/server-communications/:id", m.hdl.GetServerCommunication)

		// Video
		video := g.Group("/video")
		{
			video.POST("/generate", m.videoHdl.Generate)
			video.GET("/status/:taskId", m.videoHdl.GetStatus)
			video.DELETE("/task/:taskId", m.videoHdl.CancelTask)
			video.POST("/preview", m.videoHdl.PreviewPayload)
		}

		// Image
		img := g.Group("/image")
		{
			img.POST("/generate", m.imageHdl.Generate)
			img.GET("/status/:taskId", m.imageHdl.GetStatus)
			img.DELETE("/task/:taskId", m.imageHdl.CancelTask)
			img.POST("/preview", m.imageHdl.PreviewPayload)
		}

		// Audio
		audio := g.Group("/audio")
		{
			audio.POST("/generate", m.audioHdl.Generate)
			audio.GET("/status/:taskId", m.audioHdl.GetStatus)
			audio.DELETE("/task/:taskId", m.audioHdl.CancelTask)
			audio.POST("/preview", m.audioHdl.PreviewPayload)
		}

		// Text
		text := g.Group("/text")
		{
			text.POST("/generate", m.textHdl.Generate)
			text.GET("/status/:taskId", m.textHdl.GetStatus)
			text.DELETE("/task/:taskId", m.textHdl.CancelTask)
			text.POST("/preview", m.textHdl.PreviewPayload)

			claude := text.Group("/claude")
			{
				claude.POST("/generate-shots", m.textHdl.ClaudeGenerateShots)
				claude.POST("/generate-shots-v2", m.textHdl.ClaudeGenerateShotsV2)
				claude.GET("/generate-shots/status/:taskId", m.textHdl.GetClaudeShotsStatus)
				claude.POST("/refine-shots", m.textHdl.ClaudeRefineShots)
				claude.POST("/optimize-prompt", m.textHdl.ClaudeOptimizePrompt)
				claude.GET("/generate-shots-logs", m.textHdl.ListGenerateShotsLogs)
				claude.GET("/generate-shots-logs/:id", m.textHdl.GetGenerateShotsLog)
			}
		}
	}
}
