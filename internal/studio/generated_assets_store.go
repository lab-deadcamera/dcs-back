package studio

import (
	"database/sql"
	"fmt"
	"time"
)

// GeneratedAsset representa un recurso generado por un modelo (video, imagen, etc.).
type GeneratedAsset struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"task_id"`
	ModelName   string     `json:"model_name"`
	UserID      *int       `json:"user_id,omitempty"`
	ProjectID   string     `json:"project_id,omitempty"`
	SceneID     string     `json:"scene_id,omitempty"`
	SceneCode   string     `json:"scene_code,omitempty"`
	TakeNumber  int        `json:"take_number"`
	OriginalURL string     `json:"original_url"`
	LocalPath   string     `json:"local_path,omitempty"`
	Filename    string     `json:"filename,omitempty"`
	MimeType    string     `json:"mime_type,omitempty"`
	FileSize    int64      `json:"file_size"`
	Status      string     `json:"status"` // pending, confirmed, failed
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type GeneratedAssetStore struct {
	db  *sql.DB
	dir string // directorio local donde se guardan los assets confirmados
}

func NewGeneratedAssetStore(db *sql.DB, outputDir string) *GeneratedAssetStore {
	return &GeneratedAssetStore{db: db, dir: outputDir}
}

const genAssetCols = `id, task_id, COALESCE(model_name, '') AS model_name,
	user_id, COALESCE(project_id, '') AS project_id, COALESCE(scene_id, '') AS scene_id,
	COALESCE(scene_code, '') AS scene_code, COALESCE(take_number, 0) AS take_number,
	original_url, COALESCE(local_path, '') AS local_path,
	COALESCE(filename, '') AS filename, COALESCE(mime_type, '') AS mime_type,
	COALESCE(file_size, 0) AS file_size, status,
	confirmed_at, created_at, updated_at, deleted_at`

func (s *GeneratedAssetStore) scanAsset(a *GeneratedAsset, scanner interface{ Scan(dest ...interface{}) error }) error {
	return scanner.Scan(
		&a.ID, &a.TaskID, &a.ModelName,
		&a.UserID, &a.ProjectID, &a.SceneID, &a.SceneCode, &a.TakeNumber,
		&a.OriginalURL, &a.LocalPath,
		&a.Filename, &a.MimeType, &a.FileSize, &a.Status,
		&a.ConfirmedAt, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt,
	)
}

// Create inserta un nuevo asset generado (status: pending).
func (s *GeneratedAssetStore) Create(a *GeneratedAsset) error {
	query := `INSERT INTO generated_assets (task_id, model_name, user_id, project_id, scene_id, scene_code, take_number, original_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	return s.db.QueryRow(query,
		a.TaskID, a.ModelName, a.UserID,
		a.ProjectID, a.SceneID, a.SceneCode, a.TakeNumber,
		a.OriginalURL, a.Status,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// ListBySession devuelve los assets de un proyecto/escena, ordenado por created_at DESC.
func (s *GeneratedAssetStore) ListBySession(projectID, sceneID string) ([]GeneratedAsset, error) {
	query := `SELECT ` + genAssetCols + ` FROM generated_assets
		WHERE deleted_at IS NULL AND project_id = $1 AND scene_id = $2
		ORDER BY created_at DESC`
	rows, err := s.db.Query(query, projectID, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []GeneratedAsset
	for rows.Next() {
		var a GeneratedAsset
		if err := s.scanAsset(&a, rows); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

// ListPendingByTask devuelve los assets pendientes de una tarea.
func (s *GeneratedAssetStore) ListPendingByTask(taskID string) ([]GeneratedAsset, error) {
	query := `SELECT ` + genAssetCols + ` FROM generated_assets
		WHERE deleted_at IS NULL AND task_id = $1 AND status = 'pending'
		ORDER BY created_at DESC`
	rows, err := s.db.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []GeneratedAsset
	for rows.Next() {
		var a GeneratedAsset
		if err := s.scanAsset(&a, rows); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

// Confirm actualiza un asset como confirmado con la ruta local.
func (s *GeneratedAssetStore) Confirm(id, localPath, filename, mimeType string, fileSize int64) error {
	query := `UPDATE generated_assets
		SET status = 'confirmed', local_path = $1, filename = $2, mime_type = $3, file_size = $4, confirmed_at = NOW(), updated_at = NOW()
		WHERE id = $5 AND deleted_at IS NULL`
	result, err := s.db.Exec(query, localPath, filename, mimeType, fileSize, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("generated asset not found")
	}
	return nil
}

// Fail marca un asset como fallido.
func (s *GeneratedAssetStore) Fail(id string) error {
	_, err := s.db.Exec(`UPDATE generated_assets SET status = 'failed', updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}
