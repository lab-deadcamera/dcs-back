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
	"dcs-back-v0/internal/studio/generators"

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

	fileStore, err := file.NewStore(database, cfg.UploadDir)
	if err != nil {
		log.Fatalf("failed to init file store: %v", err)
	}
	fileSvc := file.NewService(fileStore, cfg.BaseURL)
	fileHdl := file.NewHandler(fileSvc)
	fileSvc.StartPurgeCron()

	assetSyncStore := studio.NewAssetSyncStore(database)

	charStore := character.NewStore(database)
	charSvc := character.NewService(charStore, cfg.BaseURL)
	// Enricher: attaches synced model info to character files
	charSvc.SetFileEnricher(func(files []character.CharacterFile) {
		fileIDs := make([]string, len(files))
		for i, f := range files {
			fileIDs[i] = f.FileID
		}
		syncMap, err := assetSyncStore.GetByFileIDs(fileIDs)
		if err != nil {
			return
		}
		for i, f := range files {
			assets := syncMap[f.FileID]
			if len(assets) == 0 {
				continue
			}
			seen := make(map[string]bool)
			for _, a := range assets {
				if seen[a.ModelID] {
					continue
				}
				seen[a.ModelID] = true
				m, _ := providerStore.GetModelByID(a.ModelID)
				name := "unknown"
				if m != nil {
					name = m.Name
				}
				files[i].SyncedModels = append(files[i].SyncedModels, character.SyncModelItem{
					ModelID: a.ModelID,
					Name:    name,
				})
			}
		}
	})
	charHdl := character.NewHandler(charSvc)

	studioSvc := studio.NewService(providerStore, fileSvc, cfg.OutputsDir, cfg.BaseURL)
	studioSvc.SetAssetSyncStore(assetSyncStore)
	studioSvc.SetCharacterService(charSvc)
	// Set up generation log store (request + AI response logs in DB)
	studioSvc.SetLogStore(studio.NewGenerationLogStore(database))
	// Register legacy handlers (keep for backward compat)
	studioSvc.RegisterHandler(studio.NewSeedanceHandler(cfg.OutputsDir))
	studioSvc.RegisterHandler(studio.NewSeedreamHandler(cfg.OutputsDir))
	// Register new generators
	studioSvc.RegisterGenerator(generators.NewSeedanceGenerator(cfg.OutputsDir))
	studioSvc.RegisterGenerator(generators.NewSeedreamGenerator(cfg.OutputsDir))
	studioHdl := studio.NewHandler(studioSvc)

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
			// New unified payload endpoint
			studioGroup.POST("/generate", studioHdl.GenerateUnified)
			// Legacy Selection-based endpoint
			studioGroup.POST("/generate-legacy", studioHdl.Generate)
			// Status — returns StudioStatusResponse (list of outputs)
			studioGroup.GET("/status/:taskId", studioHdl.GetStatus)
			// Legacy status format
			studioGroup.GET("/status-legacy/:taskId", studioHdl.GetStatusLegacy)
			studioGroup.DELETE("/task/:taskId", studioHdl.CancelTask)
			// Asset sync — upload file to model's asset library
			studioGroup.POST("/sync-asset", studioHdl.SyncAsset)
			studioGroup.GET("/synced-assets", studioHdl.ListSyncedAssets)
			// Sync-aware file listing (includes synced_models)
			studioGroup.GET("/files-with-sync", studioHdl.ListFilesWithSync)
			// Character files with sync info
			studioGroup.GET("/characters/:id/files-with-sync", studioHdl.ListCharacterFilesWithSync)
			// Sync all character assets to a model
			studioGroup.POST("/sync-character-assets", studioHdl.SyncCharacterAssets)
			// Generation log CRUD (no delete)
			studioGroup.GET("/logs/generation", studioHdl.ListGenerationLogs)
			studioGroup.GET("/logs/generation/:id", studioHdl.GetGenerationLog)
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
