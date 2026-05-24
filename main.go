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
	"dcs-back-v0/internal/module"
	"dcs-back-v0/internal/project"
	"dcs-back-v0/internal/provider"
	"dcs-back-v0/internal/studio"
	studioaudio "dcs-back-v0/internal/studio/audio"
	studioimage "dcs-back-v0/internal/studio/image"
	studioimagegens "dcs-back-v0/internal/studio/image/generators"
	studiotext "dcs-back-v0/internal/studio/text"
	studiovideo "dcs-back-v0/internal/studio/video"
	videogens "dcs-back-v0/internal/studio/video/generators"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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

	// ─── Init stores & services ──────────────────────────────────

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
	studioSvc.SetAssetCredentials(cfg.AssetAccessKeyID, cfg.AssetSecretAccessKey, cfg.AssetDefaultGroupID)
	studioSvc.SetLogStore(studio.NewGenerationLogStore(database))
	studioSvc.SetCommStore(studio.NewServerCommunicationStore(database))
	studioSvc.RegisterHandler(studio.NewSeedanceHandler(cfg.OutputsDir))
	studioSvc.RegisterHandler(studio.NewSeedreamHandler(cfg.OutputsDir))
	studioSvc.RegisterGenerator(videogens.NewSeedanceGenerator(cfg.OutputsDir))
	studioSvc.RegisterGenerator(videogens.NewSeedanceGalleryGenerator(cfg.OutputsDir))
	studioSvc.RegisterGenerator(studioimagegens.NewSeedreamGenerator(cfg.OutputsDir))
	studioSvc.RegisterGenerator(studioimagegens.NewGeminiNanoGenerator(cfg.OutputsDir))
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

	// ─── Router ──────────────────────────────────────────────────

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
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	r.Static("/outputs", cfg.OutputsDir)
	r.Static("/docs", "./docs")

	// ─── Module registration ─────────────────────────────────────

	v1 := r.Group("/api/v1")
	authMw := middleware.Auth(cfg.JWTSecret)
	adminMw := middleware.RequireRole(1)

	registry := module.NewRegistry()
	registry.Register(auth.NewModule(authHdl))
	registry.Register(image.NewModule(imageHdl))
	registry.Register(file.NewModule(fileHdl))
	registry.Register(character.NewModule(charHdl))
	registry.Register(provider.NewModule(providerHdl))
	registry.Register(project.NewModule(projectHdl))
	registry.Register(studio.NewModule(studioHdl, studioVideoHdl, studioImageHdl, studioAudioHdl, studioTextHdl))
	registry.Setup(v1, authMw, adminMw)

	// ─── Start ───────────────────────────────────────────────────

	log.Printf("server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
