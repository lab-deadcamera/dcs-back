package config

import "os"

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
}

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

	return &Config{
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
		SuperAdminUsername: superAdminUsername,
		SuperAdminPassword: superAdminPassword,
		SuperAdminName:     os.Getenv("SUPER_ADMIN_NAME"),
		SuperAdminSurname:  os.Getenv("SUPER_ADMIN_SURNAME"),
		SuperAdminUserName: os.Getenv("SUPER_ADMIN_USER_NAME"),
		SuperAdminEmail:    os.Getenv("SUPER_ADMIN_EMAIL"),
		AssetAccessKeyID:     assetAccessKeyID,
		AssetSecretAccessKey: assetSecretAccessKey,
		AssetDefaultGroupID:  assetDefaultGroupID,
	}
}
