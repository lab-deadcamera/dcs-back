package utils


import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ModeBinary = "binary"
	ModeURL    = "url"
	ModeBase64 = "base64"

	MediaTypeVideo = "video"
	MediaTypeAudio = "audio"
	MediaTypeImage = "image"
	MediaTypeOther = "other"
)

// DownloadFromURL downloads content from a URL and returns raw bytes.
func DownloadFromURL(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("URL returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// DecodeBase64 decodes a base64 string, stripping any data URI prefix if present
// (e.g. "data:image/png;base64,iVBOR...").
func DecodeBase64(s string) ([]byte, error) {
	if idx := strings.Index(s, "base64,"); idx != -1 {
		s = s[idx+7:]
	}

	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty base64 data")
	}

	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 data: %w", err)
	}

	return data, nil
}

// DetectMediaType detects the media type from a filename extension.
// Returns "video", "audio", "image", or "other".
func DetectMediaType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv", ".wmv", ".m4v", ".3gp", ".ts":
		return MediaTypeVideo
	case ".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a", ".wma", ".opus", ".aiff":
		return MediaTypeAudio
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".tiff", ".tif", ".ico":
		return MediaTypeImage
	default:
		return MediaTypeOther
	}
}

// SaveToOutput saves raw bytes to the outputs directory.
// Uses DetectMediaType to place files in a subfolder: /outputs/{video,audio,image,other}/
// Returns the public URL path (e.g. "/outputs/video/filename.mp4").
// Creates the directory tree if it does not exist.
func SaveToOutput(data []byte, filename, outputsDir string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("cannot save empty data")
	}

	filename = sanitizeFilename(filename)
	if filename == "" {
		return "", fmt.Errorf("invalid filename after sanitization")
	}

	mediaType := DetectMediaType(filename)
	subDir := filepath.Join(outputsDir, mediaType)

	fullPath := filepath.Join(subDir, filename)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", subDir, err)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return "/outputs/" + mediaType + "/" + filename, nil
}

// SaveBinaryOutput saves raw binary data to the outputs directory.
// Shorthand for SaveToOutput with pre-validated []byte.
func SaveBinaryOutput(data []byte, filename, outputsDir string) (string, error) {
	return SaveToOutput(data, filename, outputsDir)
}

// SaveURLOutput downloads a URL and saves its content to the outputs directory.
func SaveURLOutput(rawURL, filename, outputsDir string) (string, error) {
	data, err := DownloadFromURL(rawURL)
	if err != nil {
		return "", err
	}
	return SaveToOutput(data, filename, outputsDir)
}

// SaveBase64Output decodes base64 data and saves it to the outputs directory.
func SaveBase64Output(b64Data, filename, outputsDir string) (string, error) {
	data, err := DecodeBase64(b64Data)
	if err != nil {
		return "", err
	}
	return SaveToOutput(data, filename, outputsDir)
}

// SaveOutput handles all three input modes in one call.
// Supported modes: "binary" (input must be []byte), "url" (string), "base64" (string).
func SaveOutput(input interface{}, mode, filename, outputsDir string) (string, error) {
	switch mode {
	case ModeBinary:
		data, ok := input.([]byte)
		if !ok {
			return "", fmt.Errorf("invalid input type for mode %q: expected []byte", mode)
		}
		return SaveBinaryOutput(data, filename, outputsDir)

	case ModeURL:
		s, ok := input.(string)
		if !ok {
			return "", fmt.Errorf("invalid input type for mode %q: expected string", mode)
		}
		return SaveURLOutput(s, filename, outputsDir)

	case ModeBase64:
		s, ok := input.(string)
		if !ok {
			return "", fmt.Errorf("invalid input type for mode %q: expected string", mode)
		}
		return SaveBase64Output(s, filename, outputsDir)

	default:
		return "", fmt.Errorf("unsupported mode: %q (use %q, %q, or %q)",
			mode, ModeBinary, ModeURL, ModeBase64)
	}
}

// sanitizeFilename removes path separators and known dangerous characters
// to prevent directory traversal.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "..", "")
	name = strings.TrimSpace(name)
	return name
}
