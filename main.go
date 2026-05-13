package main

import (
	"log"
	"os"

	"dcs-back-v0/config"
	"dcs-back-v0/internal/auth"
	"dcs-back-v0/internal/character"
	"dcs-back-v0/internal/db"
	"dcs-back-v0/internal/file"
	"dcs-back-v0/internal/image"
	"dcs-back-v0/internal/middleware"
	"dcs-back-v0/internal/studio"

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

	studioStore, err := studio.NewStore(cfg.KeysFile, cfg.PresetsFile, cfg.OutputsDir)
	if err != nil {
		log.Fatalf("failed to init studio store: %v", err)
	}
	studioSvc := studio.NewService(studioStore)
	studioHdl := studio.NewHandler(studioSvc)

	fileStore, err := file.NewStore(database, cfg.UploadDir)
	if err != nil {
		log.Fatalf("failed to init file store: %v", err)
	}
	fileSvc := file.NewService(fileStore, cfg.BaseURL)
	fileHdl := file.NewHandler(fileSvc)
	fileSvc.StartPurgeCron()

	charStore := character.NewStore(database)
	charSvc := character.NewService(charStore, cfg.BaseURL)
	charHdl := character.NewHandler(charSvc)

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "hola mundo"})
	})

	r.Static("/outputs", cfg.OutputsDir)
	r.Static("/docs", "./docs")

	origins := os.Getenv("CORS_ALLOW_ORIGINS")
	if origins == "" {
		origins = "*"
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{origins},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
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

		studioGroup := v1.Group("/")
		{
			studioGroup.GET("/keys", studioHdl.ListKeys)
			studioGroup.POST("/keys", studioHdl.AddKey)
			studioGroup.POST("/keys/:id/activate", studioHdl.ActivateKey)
			studioGroup.DELETE("/keys/:id", studioHdl.DeleteKey)
			studioGroup.PATCH("/keys/:id", studioHdl.UpdateKey)
			studioGroup.GET("/presets", studioHdl.GetPresets)
			studioGroup.POST("/compile-prompt", studioHdl.CompilePrompt)
			studioGroup.POST("/generate", studioHdl.Generate)
			studioGroup.GET("/status/:taskId", studioHdl.GetStatus)
			studioGroup.DELETE("/task/:taskId", studioHdl.CancelTask)
			studioGroup.GET("/seedream/assets", studioHdl.ListTrustedAssets)
			studioGroup.POST("/seedream/generate", studioHdl.GenerateSeedream)
			studioGroup.POST("/assets/groups", studioHdl.CreateAssetGroup)
			studioGroup.GET("/assets/groups", studioHdl.ListAssetGroups)
			studioGroup.POST("/assets", studioHdl.CreateAsset)
			studioGroup.GET("/assets/:id", studioHdl.GetAsset)
			studioGroup.GET("/assets", studioHdl.ListAssets)
			studioGroup.DELETE("/assets/:id", studioHdl.DeleteAsset)
			studioGroup.GET("/health", studioHdl.Health)
			studioGroup.GET("/debug", studioHdl.Debug)
		}

		filesAPI := v1.Group("/files")
		{
			filesAPI.POST("/upload", fileHdl.Upload)
			filesAPI.GET("/trash", fileHdl.ListTrash)
			filesAPI.GET("", fileHdl.ListFiles)
			filesAPI.GET("/:id", fileHdl.GetFile)
			filesAPI.GET("/:id/serve", fileHdl.ServeFile)
			filesAPI.GET("/:id/thumbnail", fileHdl.ServeThumbnail)
			filesAPI.DELETE("/:id", fileHdl.SoftDelete)
			filesAPI.POST("/:id/restore", fileHdl.Restore)
			filesAPI.POST("/:id/recover-temp", fileHdl.RecoverTemp)
			filesAPI.DELETE("/:id/hard", fileHdl.HardDelete)
		}

		charactersAPI := v1.Group("/characters")
		{
			charactersAPI.POST("", charHdl.Create)
			charactersAPI.GET("", charHdl.List)
			charactersAPI.GET("/:id", charHdl.GetByID)
			charactersAPI.PATCH("/:id", charHdl.Update)
			charactersAPI.DELETE("/:id", charHdl.SoftDelete)
			charactersAPI.POST("/:id/files", charHdl.AddFile)
			charactersAPI.GET("/:id/files", charHdl.ListFiles)
			charactersAPI.DELETE("/:id/files/:fileId", charHdl.RemoveFile)
		}
	}

	log.Printf("server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
