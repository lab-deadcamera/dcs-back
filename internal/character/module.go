package character

import (
	"github.com/gin-gonic/gin"
)

type Module struct {
	hdl *Handler
}

func NewModule(hdl *Handler) *Module {
	return &Module{hdl: hdl}
}

func (m *Module) Name() string { return "character" }

func (m *Module) Register(rg *gin.RouterGroup, authMw, _ gin.HandlerFunc) {
	g := rg.Group("/characters")
	g.Use(authMw)
	{
		g.POST("", m.hdl.Create)
		g.GET("", m.hdl.List)
		g.GET("/:id", m.hdl.GetByID)
		g.PATCH("/:id", m.hdl.Update)
		g.DELETE("/:id", m.hdl.SoftDelete)
		g.POST("/:id/files", m.hdl.AddFile)
		g.GET("/:id/files", m.hdl.ListFiles)
		g.DELETE("/:id/files/:fileId", m.hdl.RemoveFile)
	}
}
