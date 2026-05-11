package main

import (
	"log"
	"os"

	"dcs-back-v0/config"
	"dcs-back-v0/internal/auth"
	"dcs-back-v0/internal/db"
	"dcs-back-v0/internal/image"
	"dcs-back-v0/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró el archivo .env, usando variables de entorno")
	}

	cfg := config.Load()

	database, err := db.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer database.Close()

	authStore := auth.NewStore(database)
	authSvc := auth.NewService(authStore, cfg.JWTSecret)
	authHdl := auth.NewHandler(authSvc)

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
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", authHdl.Register)
			authGroup.POST("/login", authHdl.Login)
		}

		imagesAPI := v1.Group("/images")
		{
			// Public image routes with Rate Limiting (e.g. 10 requests per second, burst of 20)
			imagesAPI.GET("/:filename", middleware.RateLimit(rate.Limit(10), 20), h.Serve)
			imagesAPI.GET("/thumbnails/:filename", middleware.RateLimit(rate.Limit(10), 20), h.ServeThumbnail)

			// Protected routes
			protected := imagesAPI.Group("/")
			protected.Use(middleware.Auth(cfg.JWTSecret))
			{
				protected.POST("/upload", h.Upload)
				protected.GET("/list", h.List)
				protected.DELETE("/:filename", h.Delete)
			}
		}
	}

	log.Printf("server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
