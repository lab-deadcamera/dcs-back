package provider

import (
	"github.com/gin-gonic/gin"
)

type Module struct {
	hdl *Handler
}

func NewModule(hdl *Handler) *Module {
	return &Module{hdl: hdl}
}

func (m *Module) Name() string { return "provider" }

func (m *Module) Register(rg *gin.RouterGroup, authMw, _ gin.HandlerFunc) {
	// Providers
	providers := rg.Group("/providers")
	providers.Use(authMw)
	{
		providers.POST("", m.hdl.CreateProvider)
		providers.GET("", m.hdl.ListProviders)
		providers.GET("/:id", m.hdl.GetProvider)
		providers.PATCH("/:id", m.hdl.UpdateProvider)
		providers.DELETE("/:id", m.hdl.SoftDeleteProvider)
		providers.GET("/:id/models", m.hdl.ListModelsByProvider)
	}

	// Models
	models := rg.Group("/models")
	models.Use(authMw)
	{
		models.POST("", m.hdl.CreateModel)
		models.GET("", m.hdl.ListModels)
		models.GET("/:id", m.hdl.GetModel)
		models.PATCH("/:id", m.hdl.UpdateModel)
		models.GET("/favorite", m.hdl.GetFavorite)
		models.POST("/:id/favorite", m.hdl.SetFavorite)
		models.DELETE("/:id", m.hdl.SoftDeleteModel)
	}
}
