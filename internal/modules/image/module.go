package image

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"dcs-back-v0/internal/middleware"
)

type Module struct {
	hdl    *Handler
}

func NewModule(hdl *Handler) *Module {
	return &Module{hdl: hdl}
}

func (m *Module) Name() string { return "image" }

func (m *Module) Register(rg *gin.RouterGroup, authMw, _ gin.HandlerFunc) {
	// Public routes (rate-limited)
	pub := rg.Group("/images")
	{
		pub.GET("/:filename", middleware.RateLimit(rate.Limit(10), 20), m.hdl.Serve)
		pub.GET("/thumbnails/:filename", middleware.RateLimit(rate.Limit(10), 20), m.hdl.ServeThumbnail)
	}

	// Protected routes
	priv := rg.Group("/images")
	priv.Use(authMw)
	{
		priv.POST("/upload", m.hdl.Upload)
		priv.GET("/list", m.hdl.List)
		priv.DELETE("/:filename", m.hdl.Delete)
	}
}
