package image

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	store       *Store
	thumbnailW  int
	thumbnailH  int
	baseURL     string
	allowedExts map[string]bool
	maxFileSize int64
}

type UploadResult struct {
	Filename     string `json:"filename"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Size         int64  `json:"size"`
}

func NewService(store *Store, baseURL string, thumbW, thumbH int, allowedExts map[string]bool, maxSize int64) *Service {
	return &Service{
		store:       store,
		baseURL:     baseURL,
		thumbnailW:  thumbW,
		thumbnailH:  thumbH,
		allowedExts: allowedExts,
		maxFileSize: maxSize,
	}
}

func (s *Service) Upload(fileHeader *multipart.FileHeader) (*UploadResult, error) {
	if fileHeader.Size > s.maxFileSize {
		return nil, fmt.Errorf("file too large: max %d bytes", s.maxFileSize)
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !s.allowedExts[ext] {
		return nil, fmt.Errorf("file type %s not allowed", ext)
	}

	f, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return s.saveImage(data, ext, fileHeader.Size)
}

func (s *Service) UploadBase64(originalFilename, b64Data string) (*UploadResult, error) {
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if !s.allowedExts[ext] {
		return nil, fmt.Errorf("file type %s not allowed", ext)
	}

	data, err := decodeBase64(b64Data)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 data: %w", err)
	}

	if int64(len(data)) > s.maxFileSize {
		return nil, fmt.Errorf("file too large: max %d bytes", s.maxFileSize)
	}

	return s.saveImage(data, ext, int64(len(data)))
}

func decodeBase64(s string) ([]byte, error) {
	if idx := strings.Index(s, "base64,"); idx != -1 {
		s = s[idx+7:]
	}
	return base64.StdEncoding.DecodeString(s)
}

func (s *Service) saveImage(data []byte, ext string, size int64) (*UploadResult, error) {
	filename := uuid.New().String() + ext

	if err := s.store.Save(filename, data); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	if err := s.store.GenerateThumbnail(filename, s.thumbnailW, s.thumbnailH); err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return &UploadResult{
		Filename:     filename,
		URL:          s.baseURL + "/api/v1/images/" + filename,
		ThumbnailURL: s.baseURL + "/api/v1/images/thumbnails/" + filename,
		Size:         size,
	}, nil
}

func (s *Service) Delete(filename string) error {
	return s.store.Delete(filename)
}

func (s *Service) GetPath(filename string) string {
	return s.store.GetPath(filename)
}

func (s *Service) GetThumbnailPath(filename string) string {
	return s.store.GetThumbnailPath(filename)
}

func (s *Service) List() ([]string, error) {
	return s.store.List()
}
