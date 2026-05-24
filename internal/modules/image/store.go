package image

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"golang.org/x/image/webp"
)

type Store struct {
	UploadDir    string
	ThumbnailDir string
}

func NewStore(uploadDir, thumbnailDir string) (*Store, error) {
	for _, dir := range []string{uploadDir, thumbnailDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return &Store{UploadDir: uploadDir, ThumbnailDir: thumbnailDir}, nil
}

func (s *Store) Save(filename string, data []byte) error {
	return os.WriteFile(filepath.Join(s.UploadDir, filename), data, 0644)
}

func (s *Store) Delete(filename string) error {
	orig := filepath.Join(s.UploadDir, filename)
	thumb := filepath.Join(s.ThumbnailDir, filename)

	os.Remove(thumb)
	return os.Remove(orig)
}

func (s *Store) GetPath(filename string) string {
	return filepath.Join(s.UploadDir, filename)
}

func (s *Store) GetThumbnailPath(filename string) string {
	return filepath.Join(s.ThumbnailDir, filename)
}

func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.UploadDir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

func (s *Store) GenerateThumbnail(filename string, width, height int) error {
	srcPath := filepath.Join(s.UploadDir, filename)
	dstPath := filepath.Join(s.ThumbnailDir, filename)

	src, err := decodeImage(srcPath)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	dst := imaging.Fit(src, width, height, imaging.Lanczos)

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return imaging.Save(dst, dstPath, imaging.JPEGQuality(80))
	case ".png":
		return imaging.Save(dst, dstPath, imaging.PNGCompressionLevel(6))
	case ".gif":
		return imaging.Save(dst, dstPath)
	case ".webp":
		return imaging.Save(dst, dstPath)
	default:
		return imaging.Save(dst, dstPath)
	}
}

func decodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Decode(f)
	case ".png":
		return png.Decode(f)
	case ".gif":
		return gif.Decode(f)
	case ".webp":
		return webp.Decode(f)
	default:
		return imaging.Open(path)
	}
}
