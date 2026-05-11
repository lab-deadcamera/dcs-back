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
	KeysFile        string
	PresetsFile     string
	OutputsDir      string
	DefaultModel    string
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

	keysFile := os.Getenv("KEYS_FILE")
	if keysFile == "" {
		keysFile = "./keys.json"
	}

	presetsFile := os.Getenv("PRESETS_FILE")
	if presetsFile == "" {
		presetsFile = "./presets.json"
	}

	outputsDir := os.Getenv("OUTPUTS_DIR")
	if outputsDir == "" {
		outputsDir = "./outputs"
	}

	defaultModel := os.Getenv("DEFAULT_MODEL")
	if defaultModel == "" {
		defaultModel = "dreamina-seedance-2-0-fast-260128"
	}

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
		KeysFile:        keysFile,
		PresetsFile:     presetsFile,
		OutputsDir:      outputsDir,
		DefaultModel:    defaultModel,
		AllowedExts: map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".webp": true,
		},
	}
}
