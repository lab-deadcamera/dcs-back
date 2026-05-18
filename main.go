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
	"dcs-back-v0/internal/project"
	"dcs-back-v0/internal/provider"
	"dcs-back-v0/internal/studio"
	studioaudio "dcs-back-v0/internal/studio/audio"
	studioimage "dcs-back-v0/internal/studio/image"
	studiotext "dcs-back-v0/internal/studio/text"
	studiovideo "dcs-back-v0/internal/studio/video"
	videogens "dcs-back-v0/internal/studio/video/generators"

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
	authSvc.SetSuperAdminConfig(cfg.SuperAdminUsername, cfg.SuperAdminPassword, cfg.SuperAdminName, cfg.SuperAdminSurname, cfg.SuperAdminUserName, cfg.SuperAdminEmail)
	if err := authSvc.SeedSuperAdmin(); err != nil {
		log.Printf("warning: super admin seed: %v", err)
	}
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
	studioSvc.SetLogStore(studio.NewGenerationLogStore(database))
	studioSvc.SetCommStore(studio.NewServerCommunicationStore(database))
	studioSvc.RegisterHandler(studio.NewSeedanceHandler(cfg.OutputsDir))
	studioSvc.RegisterHandler(studio.NewSeedreamHandler(cfg.OutputsDir))
	studioSvc.RegisterGenerator(videogens.NewSeedanceGenerator(cfg.OutputsDir))
						studioSvc.RegisterGenerator(videogens.NewSeedanceGalleryGenerator(cfg.OutputsDir))
	studioSvc.RegisterGenerator(studioimage.NewSeedreamGenerator(cfg.OutputsDir))
	studioHdl := studio.NewHandler(studioSvc)

	vidSvc := studiovideo.NewService(studioSvc)
	studioVideoHdl := studiovideo.NewHandler(vidSvc)
	imgSvc := studioimage.NewService(studioSvc)
	projectStore := project.NewStore(database)
	studioImageHdl := studioimage.NewHandler(imgSvc)
	studioAudioHdl := studioaudio.NewHandler(studioSvc)
	studioTextHdl := studiotext.NewHandler(studioSvc)
	projectSvc := project.NewService(projectStore)
	projectHdl := project.NewHandler(projectSvc)

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
			authGroup.GET("/profile", middleware.Auth(cfg.JWTSecret), authHdl.GetProfile)
		}

		adminGroup := v1.Group("/admin")
		adminGroup.Use(middleware.Auth(cfg.JWTSecret), middleware.RequireRole(1))
		{
			adminGroup.POST("/users", authHdl.CreateUser)
			adminGroup.GET("/users", authHdl.ListUsers)
			adminGroup.GET("/roles", authHdl.ListRoles)
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

			studioGroup.POST("/generate-legacy", studioHdl.Generate)

			studioGroup.GET("/status-legacy/:taskId", studioHdl.GetStatusLegacy)

			studioGroup.POST("/sync-asset", studioHdl.SyncAsset)
			studioGroup.GET("/synced-assets", studioHdl.ListSyncedAssets)
			studioGroup.GET("/files-with-sync", studioHdl.ListFilesWithSync)
			studioGroup.GET("/characters/:id/files-with-sync", studioHdl.ListCharacterFilesWithSync)
			studioGroup.POST("/sync-character-assets", studioHdl.SyncCharacterAssets)
			studioGroup.GET("/logs/generation", studioHdl.ListGenerationLogs)
			studioGroup.GET("/logs/generation/:id", studioHdl.GetGenerationLog)
			studioGroup.GET("/logs/server-communications", studioHdl.ListServerCommunications)
			studioGroup.GET("/logs/server-communications/:id", studioHdl.GetServerCommunication)

			videoGroup := studioGroup.Group("/video")
			videoGroup.POST("/generate", studioVideoHdl.Generate)
			videoGroup.GET("/status/:taskId", studioVideoHdl.GetStatus)
			videoGroup.DELETE("/task/:taskId", studioVideoHdl.CancelTask)
			videoGroup.POST("/preview", studioVideoHdl.PreviewPayload)

			imageGroup := studioGroup.Group("/image")
			imageGroup.POST("/generate", studioImageHdl.Generate)
			imageGroup.GET("/status/:taskId", studioImageHdl.GetStatus)
			imageGroup.DELETE("/task/:taskId", studioImageHdl.CancelTask)
			imageGroup.POST("/preview", studioImageHdl.PreviewPayload)

			audioGroup := studioGroup.Group("/audio")
			audioGroup.POST("/generate", studioAudioHdl.Generate)
			audioGroup.GET("/status/:taskId", studioAudioHdl.GetStatus)
			audioGroup.DELETE("/task/:taskId", studioAudioHdl.CancelTask)
			audioGroup.POST("/preview", studioAudioHdl.PreviewPayload)

			textGroup := studioGroup.Group("/text")
			textGroup.POST("/generate", studioTextHdl.Generate)
			textGroup.GET("/status/:taskId", studioTextHdl.GetStatus)
			textGroup.DELETE("/task/:taskId", studioTextHdl.CancelTask)
			textGroup.POST("/preview", studioTextHdl.PreviewPayload)

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
			projectsAPI := v1.Group("/projects")
			{
				projectsAPI.POST("", projectHdl.Create)
				projectsAPI.GET("", projectHdl.List)
				projectsAPI.GET("/:id", projectHdl.GetByID)
				projectsAPI.PATCH("/:id", projectHdl.Update)
				projectsAPI.DELETE("/:id", projectHdl.SoftDelete)
				projectsAPI.POST("/:id/scenes", projectHdl.CreateScene)
				projectsAPI.GET("/:id/scenes", projectHdl.ListScenes)
				projectsAPI.GET("/:id/scenes/:sceneId", projectHdl.GetSceneByID)
				projectsAPI.PATCH("/:id/scenes/:sceneId", projectHdl.UpdateScene)
				projectsAPI.DELETE("/:id/scenes/:sceneId", projectHdl.SoftDeleteScene)
				projectsAPI.POST("/:id/scenes/:sceneId/takes", projectHdl.CreateTake)
				projectsAPI.GET("/:id/scenes/:sceneId/takes", projectHdl.ListTakes)
				projectsAPI.GET("/:id/scenes/:sceneId/takes/:takeId", projectHdl.GetTakeByID)
				projectsAPI.PATCH("/:id/scenes/:sceneId/takes/:takeId", projectHdl.UpdateTake)
				projectsAPI.DELETE("/:id/scenes/:sceneId/takes/:takeId", projectHdl.SoftDeleteTake)
				projectsAPI.POST("/:id/scenes/:sceneId/takes/save-generation", projectHdl.SaveGeneration)
				projectsAPI.POST("/:id/scenes/:sceneId/takes/:takeId/toggle-active", projectHdl.ToggleTakeActive)
			}
		}
	}

	log.Printf("server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
