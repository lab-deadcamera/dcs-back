package config

import (
	"log"
	"os"
)

type Config struct {
	Port            string
	UploadDir       string
	ThumbnailDir    string
	MaxFileSize     int64
	ThumbnailWidth  int
	ThumbnailHeight int
	BaseURL         string
	AllowedExts     map[string]bool
	DatabaseURL     string
	JWTSecret       string
	OutputsDir      string

	// Super admin seed
	SuperAdminUsername string
	SuperAdminPassword string
	SuperAdminName     string
	SuperAdminSurname  string
	SuperAdminUserName string
	SuperAdminEmail    string

	// BytePlus asset library credentials (gallery sync)
	AssetAccessKeyID     string
	AssetSecretAccessKey string
	AssetDefaultGroupID  string

	// Asset fix on BytePlus validation errors
	AssetAutoNormalize bool   // redimensionar/pad/crop imágenes fuera de rango antes de subir
	AssetAspectFix     string // "pad" (barras, conserva contenido) | "crop" (recorta)
	AssetAIRepair      bool   // reparar con IA en el retry si la normalización no basta
	AssetImageModel    string // modelo de imagen usado por el repair con IA
}

var AppConfig *Config

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9099"
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	baseURL := os.Getenv("URL_PUBLIC")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://dcs:dcs_pass@localhost:5432/dcs_db?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super_secret_jwt_key_development_only"
	}

	outputsDir := os.Getenv("OUTPUTS_DIR")
	if outputsDir == "" {
		outputsDir = "./outputs"
	}

	superAdminUsername := os.Getenv("SUPER_ADMIN_USERNAME")
	if superAdminUsername == "" {
		superAdminUsername = "superadmin"
	}
	superAdminPassword := os.Getenv("SUPER_ADMIN_PASSWORD")
	if superAdminPassword == "" {
		superAdminPassword = "superadmin_pass_123"
	}

	assetAccessKeyID := os.Getenv("ASSET_ACCESS_KEY_ID")
	assetSecretAccessKey := os.Getenv("ASSET_SECRET_ACCESS_KEY")
	assetDefaultGroupID := os.Getenv("ASSET_DEFAULT_GROUP_ID")

	assetAutoNormalize := os.Getenv("ASSET_AUTO_NORMALIZE")
	if assetAutoNormalize == "" {
		assetAutoNormalize = "true"
	}
	assetAspectFix := os.Getenv("ASSET_ASPECT_FIX")
	if assetAspectFix == "" {
		assetAspectFix = "pad"
	}
	assetAIRepair := os.Getenv("ASSET_AI_REPAIR")
	if assetAIRepair == "" {
		assetAIRepair = "true"
	}
	assetImageModel := os.Getenv("ASSET_IMAGE_MODEL")
	if assetImageModel == "" {
		assetImageModel = "dreamina-seedream-4-pro-251224"
	}

	log.Printf("[config] ASSET_ACCESS_KEY_ID=%s", tern(assetAccessKeyID != "", "set", "EMPTY"))
	log.Printf("[config] ASSET_SECRET_ACCESS_KEY=%s", tern(assetSecretAccessKey != "", "set", "EMPTY"))
	log.Printf("[config] ASSET_DEFAULT_GROUP_ID=%s", tern(assetDefaultGroupID != "", "set", "EMPTY"))

	AppConfig = &Config{
		Port:            port,
		UploadDir:       uploadDir,
		ThumbnailDir:    uploadDir + "/thumbnails",
		MaxFileSize:     10 << 20,
		ThumbnailWidth:  300,
		ThumbnailHeight: 300,
		BaseURL:         baseURL,
		DatabaseURL:     databaseURL,
		JWTSecret:       jwtSecret,
		OutputsDir:      outputsDir,
		AllowedExts: map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".webp": true,
		},
		SuperAdminUsername:   superAdminUsername,
		SuperAdminPassword:   superAdminPassword,
		SuperAdminName:       os.Getenv("SUPER_ADMIN_NAME"),
		SuperAdminSurname:    os.Getenv("SUPER_ADMIN_SURNAME"),
		SuperAdminUserName:   os.Getenv("SUPER_ADMIN_USER_NAME"),
		SuperAdminEmail:      os.Getenv("SUPER_ADMIN_EMAIL"),
		AssetAccessKeyID:     assetAccessKeyID,
		AssetSecretAccessKey: assetSecretAccessKey,
		AssetDefaultGroupID:  assetDefaultGroupID,

		AssetAutoNormalize: assetAutoNormalize == "true",
		AssetAspectFix:     assetAspectFix,
		AssetAIRepair:      assetAIRepair == "true",
		AssetImageModel:    assetImageModel,
	}
	return AppConfig
}

// OutPutUrl returns the URL path prefix for serving output files.
func OutPutUrl() string {
	return "/outputs"
}

func tern(ok bool, a, b string) string {
	if ok {
		return a
	}
	return b
}
