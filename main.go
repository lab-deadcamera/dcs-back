package main

import (
	"log"
	"os"

	"dcs-back-v0/config"
	"dcs-back-v0/internal/db"
	"dcs-back-v0/internal/middleware"
	"dcs-back-v0/internal/modules"
	"dcs-back-v0/internal/modules/auth"
	"dcs-back-v0/internal/modules/character"
	"dcs-back-v0/internal/modules/file"
	"dcs-back-v0/internal/modules/image"
	"dcs-back-v0/internal/modules/preset"
	"dcs-back-v0/internal/modules/project"
	"dcs-back-v0/internal/modules/provider"
	"dcs-back-v0/internal/modules/studio"
	studioaudio "dcs-back-v0/internal/modules/studio/audio"
	calculators "dcs-back-v0/internal/modules/studio/calculators"
	studioimage "dcs-back-v0/internal/modules/studio/image"
	studioimagegens "dcs-back-v0/internal/modules/studio/image/generators"
	studiotext "dcs-back-v0/internal/modules/studio/text"
	studiovideo "dcs-back-v0/internal/modules/studio/video"
	videogens "dcs-back-v0/internal/modules/studio/video/generators"

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
	presetStore := preset.NewStore(database)
	presetSvc := preset.NewService(presetStore)
	presetHdl := preset.NewHandler(presetSvc)

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
	genLogStore := studio.NewGenerationLogStore(database)
	studioSvc.SetLogStore(genLogStore)
	studioSvc.SetCommStore(studio.NewServerCommunicationStore(database))
	seedanceGen := videogens.NewSeedanceGenerator()
	seedanceGen.SetLogStore(genLogStore)
	studioSvc.RegisterGenerator(seedanceGen)
	seedanceGalleryGen := videogens.NewSeedanceGalleryGenerator()
	seedanceGalleryGen.SetLogStore(genLogStore)
	studioSvc.RegisterGenerator(seedanceGalleryGen)
	studioSvc.RegisterGenerator(studioimagegens.NewSeedreamGenerator())
	studioSvc.RegisterGenerator(studioimagegens.NewGeminiNanoGenerator())
	studioSvc.RegisterGenerator(studioimagegens.NewGeminiNanoProGenerator())
	studioSvc.SetGeneratedAssetStore(studio.NewGeneratedAssetStore(database, cfg.OutputsDir))
	studioSvc.RegisterCalculator(calculators.NewSeedanceCalculator())
	studioSvc.RegisterCalculator(calculators.NewSeedreamCalculator())
	studioSvc.RegisterCalculator(calculators.NewGeminiCalculator())
	studioHdl := studio.NewHandler(studioSvc)

	vidSvc := studiovideo.NewService(studioSvc)
	studioVideoHdl := studiovideo.NewHandler(vidSvc)
	imgSvc := studioimage.NewService(studioSvc)
	projectStore := project.NewStore(database)
	studioImageHdl := studioimage.NewHandler(imgSvc)
	studioAudioHdl := studioaudio.NewHandler(studioSvc)
	studioTextHdl := studiotext.NewHandler(studioSvc)
	projectSvc := project.NewService(projectStore)
	projectSvc.SetTaskLookup(func(taskID string) string {
		sr, err := studioSvc.GetStatus(taskID)
		if err != nil || sr.Status != "succeeded" {
			return ""
		}
		return sr.LocalURL
	})
	studioSvc.SetTakeSaver(func(sceneID string, takeNumber int, videoURL, videoLocalURL string) error {
		_, err := projectSvc.SaveGeneration(sceneID, &project.SaveGenerationRequest{
			Number:        takeNumber,
			VideoURL:      videoURL,
			VideoLocalURL: videoLocalURL,
		})
		return err
	})
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

	registry := modules.NewRegistry()
	registry.Register(auth.NewModule(authHdl))
	registry.Register(image.NewModule(imageHdl))
	registry.Register(file.NewModule(fileHdl))
	registry.Register(character.NewModule(charHdl))
	registry.Register(provider.NewModule(providerHdl))
	registry.Register(preset.NewModule(presetHdl))
	registry.Register(project.NewModule(projectHdl))
	registry.Register(studio.NewModule(studioHdl, studioVideoHdl, studioImageHdl, studioAudioHdl, studioTextHdl))
	registry.Setup(v1, authMw, adminMw)

	// ─── Start ───────────────────────────────────────────────────

	log.Printf("server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
