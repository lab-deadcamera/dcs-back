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
	"dcs-back-v0/internal/provider"
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

	imageStore, err := image.NewStore(cfg.UploadDir, cfg.ThumbnailDir)
	if err != nil {
		log.Fatalf("failed to init image store: %v", err)
	}
	imageSvc := image.NewService(imageStore, cfg.BaseURL, cfg.ThumbnailWidth, cfg.ThumbnailHeight, cfg.AllowedExts, cfg.MaxFileSize)
	imageHdl := image.NewHandler(imageSvc)

	providerStore := provider.NewStore(database)
	providerSvc := provider.NewService(providerStore)
	providerHdl := provider.NewHandler(providerSvc)

	studioSvc := studio.NewService(providerStore, cfg.OutputsDir)
	studioSvc.RegisterHandler(studio.NewSeedanceHandler(cfg.OutputsDir))
	studioSvc.RegisterHandler(studio.NewSeedreamHandler(cfg.OutputsDir))
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
			imagesAPI.GET("/:filename", middleware.RateLimit(rate.Limit(10), 20), imageHdl.Serve)
			imagesAPI.GET("/thumbnails/:filename", middleware.RateLimit(rate.Limit(10), 20), imageHdl.ServeThumbnail)

			protected := imagesAPI.Group("/")
			protected.Use(middleware.Auth(cfg.JWTSecret))
			{
				protected.POST("/upload", imageHdl.Upload)
				protected.GET("/list", imageHdl.List)
				protected.DELETE("/:filename", imageHdl.Delete)
			}
		}

		studioGroup := v1.Group("/studio")
		{
			studioGroup.POST("/generate", studioHdl.Generate)
			studioGroup.GET("/status/:taskId", studioHdl.GetStatus)
			studioGroup.DELETE("/task/:taskId", studioHdl.CancelTask)
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

		providersAPI := v1.Group("/providers")
		{
			providersAPI.POST("", providerHdl.CreateProvider)
			providersAPI.GET("", providerHdl.ListProviders)
			providersAPI.GET("/:id", providerHdl.GetProvider)
			providersAPI.PATCH("/:id", providerHdl.UpdateProvider)
			providersAPI.DELETE("/:id", providerHdl.SoftDeleteProvider)
			providersAPI.GET("/:id/models", providerHdl.ListModelsByProvider)
		}

		modelsAPI := v1.Group("/models")
		{
			modelsAPI.POST("", providerHdl.CreateModel)
			modelsAPI.GET("", providerHdl.ListModels)
			modelsAPI.GET("/:id", providerHdl.GetModel)
			modelsAPI.PATCH("/:id", providerHdl.UpdateModel)
			modelsAPI.GET("/favorite", providerHdl.GetFavorite)
			modelsAPI.POST("/:id/favorite", providerHdl.SetFavorite)
			modelsAPI.DELETE("/:id", providerHdl.SoftDeleteModel)
		}
	}

	log.Printf("server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
