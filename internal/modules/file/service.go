package file

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	store   *Store
	baseURL string
}

func NewService(store *Store, baseURL string) *Service {
	return &Service{store: store}
}

func (s *Service) Upload(data []byte, filename, category, storage string) (*UploadResult, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return nil, fmt.Errorf("file has no extension")
	}

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	newFilename := uuid.New().String() + ext
	fullPath := fmt.Sprintf("%s/%s", category, newFilename)

	duration := float64(0)
	if strings.HasPrefix(mimeType, "video/") {
		savedPath := filepath.Join(s.store.uploadDir, fullPath)
		duration = ReadMediaDuration(savedPath)
	}

	file := &File{
		ID:       uuid.New().String(),
		Filename: filename,
		Path:     fullPath,
		Size:     int64(len(data)),
		MimeType: mimeType,
		Category: category,
		Format:   ext[1:],
		Storage:  storage,
		Duration: duration,
	}

	if err := s.store.SaveFile(data, fullPath); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	if err := s.store.CreateFile(file); err != nil {
		s.store.DeleteFile(fullPath)
		return nil, fmt.Errorf("failed to create db record: %w", err)
	}

	// Generate thumbnail eagerly for image files (regardless of storage type).
	var thumbnailURL string
	if strings.HasPrefix(mimeType, "image/") {
		if _, err := s.store.GenerateThumbnail(fullPath, 300, 300); err != nil {
			// Non-fatal: thumbnail can be generated on-demand later if needed.
			fmt.Printf("warning: failed to generate thumbnail for %s: %v\n", fullPath, err)
		} else {
			thumbnailURL = s.baseURL + "/api/v1/files/" + file.ID + "/thumbnail"
		}
	}

	format := ext[1:]
	if format == "jpeg" {
		format = "jpg"
	}

	return &UploadResult{
		ID:           file.ID,
		Filename:     filename,
		URL:          s.baseURL + "/api/v1/files/" + file.ID,
		ThumbnailURL: thumbnailURL,
		Size:         file.Size,
		MimeType:     mimeType,
		Format:       format,
		Category:     category,
	}, nil
}

func (s *Service) GetFile(id string) (*File, error) {
	return s.store.GetFileByID(id)
}

func (s *Service) SoftDelete(id string) error {
	f, err := s.store.GetFileByID(id)
	if err != nil {
		return err
	}
	if f == nil {
		return fmt.Errorf("file not found")
	}
	if f.Trashed {
		return fmt.Errorf("file already in trash")
	}

	if err := s.store.MoveFile(f.Path, "trash/"+f.Path); err != nil {
		return fmt.Errorf("failed to move to trash: %w", err)
	}

	// Also move the associated thumbnail, if any.
	thumbSubpath := "thumbnails/" + f.Path
	if s.store.FileExists(thumbSubpath) {
		_ = s.store.MoveFile(thumbSubpath, "trash/thumbnails/"+f.Path)
	}

	return s.store.SoftDeleteFile(id)
}

func (s *Service) Restore(id string) error {
	f, err := s.store.GetFileByID(id)
	if err != nil {
		return err
	}
	if f == nil {
		return fmt.Errorf("file not found")
	}
	if !f.Trashed {
		return fmt.Errorf("file is not in trash")
	}

	if err := s.store.MoveFile("trash/"+f.Path, f.Path); err != nil {
		return fmt.Errorf("failed to restore from trash: %w", err)
	}

	// Also restore the associated thumbnail, if any.
	thumbTrashPath := "trash/thumbnails/" + f.Path
	if s.store.FileExists(thumbTrashPath) {
		_ = s.store.MoveFile(thumbTrashPath, "thumbnails/"+f.Path)
	}

	return s.store.RestoreFile(id)
}

func (s *Service) RecoverTemp(id string) error {
	f, err := s.store.GetFileByID(id)
	if err != nil {
		return err
	}
	if f == nil {
		return fmt.Errorf("file not found")
	}

	if err := s.store.MoveFile("trash/"+f.Path, "temp/"+filepath.Base(f.Path)); err != nil {
		return fmt.Errorf("failed to move temp: %w", err)
	}

	// Also move the associated thumbnail, if any.
	thumbTrashPath := "trash/thumbnails/" + f.Path
	if s.store.FileExists(thumbTrashPath) {
		newThumbPath := "thumbnails/temp/" + filepath.Base(f.Path)
		_ = s.store.MoveFile(thumbTrashPath, newThumbPath)
	}

	newPath := "temp/" + filepath.Base(f.Path)
	_, err = s.store.db.Exec(`UPDATE files SET trashed = false, deleted_at = NULL, path = $1, storage = 'temp', updated_at = NOW() WHERE id = $2`, newPath, id)
	return err
}

func (s *Service) HardDelete(id string) error {
	f, err := s.store.GetFileByID(id)
	if err != nil {
		return err
	}
	if f == nil {
		return fmt.Errorf("file not found")
	}

	subpath := f.Path
	if f.Trashed {
		subpath = "trash/" + f.Path
	}

	if s.store.FileExists(subpath) {
		if err := s.store.DeleteFile(subpath); err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	// Also delete the associated thumbnail, if any.
	thumbSubpath := "thumbnails/" + f.Path
	if s.store.FileExists(thumbSubpath) {
		_ = s.store.DeleteFile(thumbSubpath)
	}

	return s.store.HardDeleteFile(id)
}

func (s *Service) GetServePath(id string) (string, error) {
	f, err := s.store.GetFileByID(id)
	if err != nil {
		return "", err
	}
	if f == nil {
		return "", fmt.Errorf("file not found")
	}
	if f.Trashed {
		return "", fmt.Errorf("file has been deleted")
	}
	return filepath.Join(s.store.uploadDir, f.Path), nil
}

func (s *Service) GetThumbnailPath(id string) (string, error) {
	f, err := s.store.GetFileByID(id)
	if err != nil {
		return "", err
	}
	if f == nil {
		return "", fmt.Errorf("file not found")
	}
	if f.Trashed {
		return "", fmt.Errorf("file has been deleted")
	}
	return s.store.GenerateThumbnail(f.Path, 300, 300)
}

func (s *Service) ListFilesPage(page, pageSize int, category, storage, search string) (*PaginatedFiles, error) {
	return s.store.ListFilesPage(page, pageSize, category, storage, search)
}

func (s *Service) ListFiles(category, storage string, trashed bool) ([]File, error) {
	return s.store.ListFiles(category, storage, trashed)
}

func (s *Service) ListTrash() ([]File, error) {
	return s.store.ListFiles("", "", true)
}

func (s *Service) PurgeExpiredTemp() error {
	files, err := s.store.ListExpiredTemp(30 * 24 * time.Hour)
	if err != nil {
		return err
	}
	for _, f := range files {
		s.store.DeleteFile(f.Path)
		// Also delete the associated thumbnail, if any.
		thumbSubpath := "thumbnails/" + f.Path
		if s.store.FileExists(thumbSubpath) {
			_ = s.store.DeleteFile(thumbSubpath)
		}
		s.store.HardDeleteFile(f.ID)
	}
	return nil
}
