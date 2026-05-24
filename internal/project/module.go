package project

import (
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
	}
}
