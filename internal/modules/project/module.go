package project

import (
	"dcs-back-v0/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Module struct {
	hdl *Handler
}

func NewModule(hdl *Handler) *Module {
	return &Module{hdl: hdl}
}

func (m *Module) Name() string { return "project" }

func (m *Module) Register(rg *gin.RouterGroup, authMw, _ gin.HandlerFunc) {
	g := rg.Group("/projects")
	g.Use(authMw)
	{
		// ── Projects ──────────────────────────────────────────
		g.POST("", m.hdl.Create)
		g.GET("", m.hdl.List)
		g.GET("/:id", m.hdl.GetByID)
		g.PATCH("/:id", m.hdl.Update)
		g.DELETE("/:id", m.hdl.SoftDelete)

		// ── Chapters (bajo /projects/:id) ─────────────────────
		chapters := g.Group("/:id/chapters")
		{
			chapters.POST("", m.hdl.CreateChapter)
			chapters.GET("", m.hdl.ListChapters)
			chapters.GET("/:chapterId", m.hdl.GetChapterByID)
			chapters.PATCH("/:chapterId", m.hdl.UpdateChapter)
			chapters.DELETE("/:chapterId", m.hdl.SoftDeleteChapter)
		}

		// ── Scenes (bajo /projects/:id/chapters/:chapterId) ──
		scenes := g.Group("/:id/chapters/:chapterId/scenes")
		{
			scenes.POST("", m.hdl.CreateScene)
			scenes.GET("", m.hdl.ListScenes)
			scenes.GET("/:sceneId", m.hdl.GetSceneByID)
			scenes.PATCH("/:sceneId", m.hdl.UpdateScene)
			scenes.DELETE("/:sceneId", m.hdl.SoftDeleteScene)
		}

		// ── Shots (bajo /projects/:id/chapters/:chapterId/scenes/:sceneId) ──
		shots := g.Group("/:id/chapters/:chapterId/scenes/:sceneId/shots")
		{
			shots.POST("", m.hdl.CreateShot)
			shots.GET("", m.hdl.ListShots)
			shots.GET("/:shotId", m.hdl.GetShotByID)
			shots.PATCH("/:shotId", m.hdl.UpdateShot)
			shots.DELETE("/:shotId", m.hdl.SoftDeleteShot)
		}

		// ── Takes (bajo /projects/:id/chapters/:chapterId/scenes/:sceneId/shots/:shotId) ──
		takes := g.Group("/:id/chapters/:chapterId/scenes/:sceneId/shots/:shotId/takes")
		{
			takes.POST("", m.hdl.CreateTake)
			takes.GET("", m.hdl.ListTakes)
			takes.GET("/:takeId", m.hdl.GetTakeByID)
			takes.PATCH("/:takeId", m.hdl.UpdateTake)
			takes.DELETE("/:takeId", m.hdl.SoftDeleteTake)
			takes.POST("/save-generation", m.hdl.SaveGeneration)
			takes.POST("/:takeId/toggle-active", m.hdl.ToggleTakeActive)
			takes.POST("/:takeId/download", m.hdl.DownloadTake)
		}

		// ── Scene Assignments (GET anyone auth'd, POST/DELETE admin+director) ──
		assignments := g.Group("/:id/chapters/:chapterId/scenes/:sceneId/assignments")
		{
			assignments.GET("", m.hdl.GetSceneAssignments)
			assignments.POST("/presets", middleware.RequireRole(2), m.hdl.AssignPresetToScene)
			assignments.POST("/characters", middleware.RequireRole(2), m.hdl.AssignCharacterToScene)
			assignments.POST("/assets", middleware.RequireRole(2), m.hdl.AssignAssetToScene)
			assignments.DELETE("/presets/:assignmentId", middleware.RequireRole(2), m.hdl.RemoveScenePreset)
			assignments.DELETE("/characters/:assignmentId", middleware.RequireRole(2), m.hdl.RemoveSceneCharacter)
			assignments.DELETE("/assets/:assignmentId", middleware.RequireRole(2), m.hdl.RemoveSceneAsset)
		}
	}
}
