package skill

import (
	"github.com/gin-gonic/gin"
)

type Module struct {
	hdl *Handler
}

func NewModule(hdl *Handler) *Module {
	return &Module{hdl: hdl}
}

func (m *Module) Name() string { return "skill" }

func (m *Module) Register(rg *gin.RouterGroup, authMw, _ gin.HandlerFunc) {
	skills := rg.Group("/skills")
	skills.Use(authMw)
	{
		skills.POST("", m.hdl.Create)
		skills.GET("", m.hdl.List)
		skills.GET("/:id", m.hdl.Get)
		skills.PATCH("/:id", m.hdl.Update)
		skills.DELETE("/:id", m.hdl.Delete)
	}
}
