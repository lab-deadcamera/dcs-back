package push

import "github.com/gin-gonic/gin"

type Module struct {
	name string
	hdl  *Handler
}

func NewModule(hdl *Handler) *Module {
	return &Module{name: "push", hdl: hdl}
}

func (m *Module) Name() string { return m.name }

func (m *Module) Register(rg *gin.RouterGroup, authMw gin.HandlerFunc, adminMw gin.HandlerFunc) {
	protected := rg.Group(m.name)
	protected.Use(authMw)
	{
		protected.POST("/subscriptions", m.hdl.Register)
		protected.DELETE("/subscriptions", m.hdl.Unregister)
	}
}
