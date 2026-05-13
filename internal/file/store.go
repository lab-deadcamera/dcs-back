package file

import (
	"database/sql"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"golang.org/x/image/webp"
)

type Store struct {
	db        *sql.DB
	uploadDir string
}

func NewStore(db *sql.DB, uploadDir string) (*Store, error) {
	for _, dir := range []string{
		uploadDir + "/images",
		uploadDir + "/videos",
		uploadDir + "/audio",
		uploadDir + "/temp",
		uploadDir + "/trash/images",
		uploadDir + "/trash/videos",
		uploadDir + "/trash/audio",
		uploadDir + "/trash/temp",
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create dir %s: %w", dir, err)
		}
	}
	return &Store{db: db, uploadDir: uploadDir}, nil
}

func (s *Store) SaveFile(data []byte, subpath string) error {
	fullPath := filepath.Join(s.uploadDir, subpath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, 0644)
}

func (s *Store) DeleteFile(subpath string) error {
	return os.Remove(filepath.Join(s.uploadDir, subpath))
}

func (s *Store) MoveFile(srcSubpath, dstSubpath string) error {
	src := filepath.Join(s.uploadDir, srcSubpath)
	dst := filepath.Join(s.uploadDir, dstSubpath)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

func (s *Store) FileExists(subpath string) bool {
	_, err := os.Stat(filepath.Join(s.uploadDir, subpath))
	return err == nil
}

func (s *Store) CopyFile(src io.Reader, subpath string) error {
	fullPath := filepath.Join(s.uploadDir, subpath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	dst, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

// ─── Thumbnails ────────────────────────────────────────────────

func (s *Store) GenerateThumbnail(srcSubpath string, width, height int) (string, error) {
	srcPath := filepath.Join(s.uploadDir, srcSubpath)
	thumbName := "thumbnails/" + srcSubpath
	dstPath := filepath.Join(s.uploadDir, thumbName)

	if _, err := os.Stat(dstPath); err == nil {
		return dstPath, nil
	}

	src, err := decodeImage(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	dst := imaging.Fit(src, width, height, imaging.Lanczos)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(srcSubpath))
	switch ext {
	case ".jpg", ".jpeg":
		err = imaging.Save(dst, dstPath, imaging.JPEGQuality(80))
	case ".png":
		err = imaging.Save(dst, dstPath, imaging.PNGCompressionLevel(6))
	default:
		err = imaging.Save(dst, dstPath)
	}
	if err != nil {
		return "", fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return dstPath, nil
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

// ─── DB operations ────────────────────────────────────────────

func (s *Store) CreateFile(f *File) error {
	query := `INSERT INTO files (id, filename, path, size, mime_type, category, format, storage)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, f.ID, f.Filename, f.Path, f.Size, f.MimeType, f.Category, f.Format, f.Storage).
		Scan(&f.CreatedAt, &f.UpdatedAt)
}

func (s *Store) GetFileByID(id string) (*File, error) {
	f := &File{}
	query := `SELECT id, filename, path, size, mime_type, category, format, storage, trashed, created_at, updated_at, deleted_at
		FROM files WHERE id = $1`
	err := s.db.QueryRow(query, id).Scan(&f.ID, &f.Filename, &f.Path, &f.Size, &f.MimeType,
		&f.Category, &f.Format, &f.Storage, &f.Trashed, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return f, nil
}

func (s *Store) ListFiles(category, storage string, trashed bool) ([]File, error) {
	query := `SELECT id, filename, path, size, mime_type, category, format, storage, trashed, created_at, updated_at, deleted_at
		FROM files WHERE trashed = $1 AND deleted_at IS NULL`
	args := []interface{}{trashed}
	argIdx := 2

	if category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}
	if storage != "" {
		query += fmt.Sprintf(" AND storage = $%d", argIdx)
		args = append(args, storage)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Filename, &f.Path, &f.Size, &f.MimeType,
			&f.Category, &f.Format, &f.Storage, &f.Trashed, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (s *Store) SoftDeleteFile(id string) error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE files SET trashed = true, deleted_at = $1, updated_at = $1 WHERE id = $2`, now, id)
	return err
}

func (s *Store) RestoreFile(id string) error {
	_, err := s.db.Exec(`UPDATE files SET trashed = false, deleted_at = NULL, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (s *Store) HardDeleteFile(id string) error {
	_, err := s.db.Exec(`DELETE FROM files WHERE id = $1`, id)
	return err
}

func (s *Store) ListExpiredTemp(maxAge time.Duration) ([]File, error) {
	query := `SELECT id, filename, path, size, mime_type, category, format, storage, trashed, created_at, updated_at, deleted_at
		FROM files WHERE storage = 'temp' AND deleted_at IS NULL AND created_at < $1`
	cutoff := time.Now().Add(-maxAge)
	rows, err := s.db.Query(query, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Filename, &f.Path, &f.Size, &f.MimeType,
			&f.Category, &f.Format, &f.Storage, &f.Trashed, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}
