package auth

import (
	"github.com/gin-gonic/gin"
)

type Module struct {
	hdl *Handler
}

func NewModule(hdl *Handler) *Module {
	return &Module{hdl: hdl}
}

func (m *Module) Name() string { return "auth" }

func (m *Module) Register(rg *gin.RouterGroup, authMw, adminMw gin.HandlerFunc) {
	// Public routes
	pub := rg.Group("/auth")
	{
		pub.POST("/register", m.hdl.Register)
		pub.POST("/login", m.hdl.Login)
	}

	// Protected routes
	priv := rg.Group("/auth")
	priv.Use(authMw)
	{
		priv.GET("/profile", m.hdl.GetProfile)
	}

	// Admin routes
	adm := rg.Group("/admin")
	adm.Use(authMw, adminMw)
	{
		adm.POST("/users", m.hdl.CreateUser)
		adm.GET("/users", m.hdl.ListUsers)
		adm.PATCH("/users/:id/active", m.hdl.UpdateUserActive)
		adm.GET("/roles", m.hdl.ListRoles)
	}
}
