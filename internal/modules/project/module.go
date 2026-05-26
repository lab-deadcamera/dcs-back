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
		g.POST("", m.hdl.Create)
		g.GET("", m.hdl.List)
		g.GET("/:id", m.hdl.GetByID)
		g.PATCH("/:id", m.hdl.Update)
		g.DELETE("/:id", m.hdl.SoftDelete)

		g.POST("/:id/scenes", m.hdl.CreateScene)
		g.GET("/:id/scenes", m.hdl.ListScenes)
		g.GET("/:id/scenes/:sceneId", m.hdl.GetSceneByID)
		g.PATCH("/:id/scenes/:sceneId", m.hdl.UpdateScene)
		g.DELETE("/:id/scenes/:sceneId", m.hdl.SoftDeleteScene)

		g.POST("/:id/scenes/:sceneId/takes", m.hdl.CreateTake)
		g.GET("/:id/scenes/:sceneId/takes", m.hdl.ListTakes)
		g.GET("/:id/scenes/:sceneId/takes/:takeId", m.hdl.GetTakeByID)
		g.PATCH("/:id/scenes/:sceneId/takes/:takeId", m.hdl.UpdateTake)
		g.DELETE("/:id/scenes/:sceneId/takes/:takeId", m.hdl.SoftDeleteTake)
		g.POST("/:id/scenes/:sceneId/takes/save-generation", m.hdl.SaveGeneration)
		g.POST("/:id/scenes/:sceneId/takes/:takeId/toggle-active", m.hdl.ToggleTakeActive)
		g.POST("/:id/scenes/:sceneId/takes/:takeId/download", m.hdl.DownloadTake)

		// Scene assignments (GET anyone auth'd, POST/DELETE admin+director)
		g.GET("/:id/scenes/:sceneId/assignments", m.hdl.GetSceneAssignments)
		g.POST("/:id/scenes/:sceneId/assignments/presets", middleware.RequireRole(2), m.hdl.AssignPresetToScene)
		g.POST("/:id/scenes/:sceneId/assignments/characters", middleware.RequireRole(2), m.hdl.AssignCharacterToScene)
		g.POST("/:id/scenes/:sceneId/assignments/assets", middleware.RequireRole(2), m.hdl.AssignAssetToScene)
		g.DELETE("/:id/scenes/:sceneId/assignments/presets/:assignmentId", middleware.RequireRole(2), m.hdl.RemoveScenePreset)
		g.DELETE("/:id/scenes/:sceneId/assignments/characters/:assignmentId", middleware.RequireRole(2), m.hdl.RemoveSceneCharacter)
		g.DELETE("/:id/scenes/:sceneId/assignments/assets/:assignmentId", middleware.RequireRole(2), m.hdl.RemoveSceneAsset)
	}
}
