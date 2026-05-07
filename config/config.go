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

	return &Config{
		Port:            port,
		UploadDir:       uploadDir,
		ThumbnailDir:    uploadDir + "/thumbnails",
		MaxFileSize:     10 << 20,
		ThumbnailWidth:  300,
		ThumbnailHeight: 300,
		BaseURL:         baseURL,
		AllowedExts: map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".webp": true,
		},
	}
}
