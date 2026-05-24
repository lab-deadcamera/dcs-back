package file

import (
	"github.com/gin-gonic/gin"
)

type Module struct {
	hdl *Handler
}

func NewModule(hdl *Handler) *Module {
	return &Module{hdl: hdl}
}

func (m *Module) Name() string { return "file" }

func (m *Module) Register(rg *gin.RouterGroup, authMw, _ gin.HandlerFunc) {
	g := rg.Group("/files")
	g.Use(authMw)
	{
		g.POST("/upload", m.hdl.Upload)
		g.GET("/trash", m.hdl.ListTrash)
		g.GET("", m.hdl.ListFiles)
		g.GET("/:id", m.hdl.GetFile)
		g.GET("/:id/serve", m.hdl.ServeFile)
		g.GET("/:id/thumbnail", m.hdl.ServeThumbnail)
		g.DELETE("/:id", m.hdl.SoftDelete)
		g.POST("/:id/restore", m.hdl.Restore)
		g.POST("/:id/recover-temp", m.hdl.RecoverTemp)
		g.DELETE("/:id/hard", m.hdl.HardDelete)
	}
}
