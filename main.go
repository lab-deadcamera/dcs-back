package main

import (
	"log"
	"os"

	"dcs-back-v0/config"
	"dcs-back-v0/internal/image"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró el archivo .env, usando variables de entorno")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9099"
	}

	cfg := config.Load()

	store, err := image.NewStore(cfg.UploadDir, cfg.ThumbnailDir)
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	svc := image.NewService(store, cfg.BaseURL, cfg.ThumbnailWidth, cfg.ThumbnailHeight, cfg.AllowedExts, cfg.MaxFileSize)
	h := image.NewHandler(svc)

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "hola mundo"})
	})

	origins := os.Getenv("CORS_ALLOW_ORIGINS")
	if origins == "" {
		origins = "*"
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{origins},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	api := r.Group("/api/v1/images")
	{
		api.POST("/upload", h.Upload)
		api.GET("/list", h.List)
		api.GET("/:filename", h.Serve)
		api.GET("/thumbnails/:filename", h.ServeThumbnail)
		api.DELETE("/:filename", h.Delete)
	}

	log.Printf("server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
