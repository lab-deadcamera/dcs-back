package preset

import (
	"dcs-back-v0/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Module struct {
	name string
	hdl  *Handler
}

func NewModule(hdl *Handler) *Module {
	return &Module{name: "presets", hdl: hdl}
}

func (m *Module) Name() string { return m.name }

func (m *Module) Register(rg *gin.RouterGroup, authMw gin.HandlerFunc, adminMw gin.HandlerFunc) {
	protected := rg.Group(m.name)
	protected.Use(authMw)
	{
		protected.GET("/groups", m.hdl.ListGroups)
		protected.POST("/groups", middleware.RequireRole(2), m.hdl.CreateGroup)
		protected.PATCH("/groups/:id", middleware.RequireRole(2), m.hdl.UpdateGroup)

		protected.GET("", m.hdl.ListPresets)
		protected.GET("/:id", m.hdl.GetPreset)
		protected.POST("", middleware.RequireRole(2), m.hdl.CreatePreset)
		protected.PATCH("/:id", middleware.RequireRole(2), m.hdl.UpdatePreset)
		protected.DELETE("/:id", middleware.RequireRole(2), m.hdl.SoftDeletePreset)
	}
}
