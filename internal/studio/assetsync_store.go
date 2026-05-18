package studio

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ModelAsset represents a file that has been synced to a model's asset library.
type ModelAsset struct {
	ID            string    `json:"id"`
	ModelID       string    `json:"model_id"`
	FileID        string    `json:"file_id"`
	AssetID       string    `json:"asset_id"`
	AssetGroupID  string    `json:"asset_group_id"`
	Status        string    `json:"status"` // "syncing", "active", "failed"
	ErrorMessage  string    `json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ─── Queries ─────────────────────────────────────────────────────

const (
	createModelAssetSQL = `INSERT INTO model_assets (id, model_id, file_id, asset_id, asset_group_id, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

	getModelAssetSQL = `SELECT id, model_id, file_id, asset_id, asset_group_id, status, error_message, created_at, updated_at
		FROM model_assets WHERE id = $1`

	getModelAssetByFileSQL = `SELECT id, model_id, file_id, asset_id, asset_group_id, status, error_message, created_at, updated_at
		FROM model_assets WHERE model_id = $1 AND file_id = $2 ORDER BY created_at DESC LIMIT 1`

	listModelAssetsSQL = `SELECT id, model_id, file_id, asset_id, asset_group_id, status, error_message, created_at, updated_at
		FROM model_assets WHERE model_id = $1 ORDER BY created_at DESC`

	updateModelAssetStatusSQL = `UPDATE model_assets SET status = $1, error_message = $2, updated_at = NOW()
		WHERE id = $3`

	deleteModelAssetSQL = `DELETE FROM model_assets WHERE id = $1`
)

// ─── Store ───────────────────────────────────────────────────────

type AssetSyncStore struct {
	db *sql.DB
}

func NewAssetSyncStore(db *sql.DB) *AssetSyncStore {
	return &AssetSyncStore{db: db}
}

func (s *AssetSyncStore) Create(ma *ModelAsset) error {
	if ma.ID == "" {
		ma.ID = uuid.New().String()
	}
	return s.db.QueryRow(createModelAssetSQL, ma.ID, ma.ModelID, ma.FileID, ma.AssetID,
		ma.AssetGroupID, ma.Status, ma.ErrorMessage).
		Scan(&ma.CreatedAt, &ma.UpdatedAt)
}

func (s *AssetSyncStore) GetByID(id string) (*ModelAsset, error) {
	ma := &ModelAsset{}
	err := s.db.QueryRow(getModelAssetSQL, id).Scan(&ma.ID, &ma.ModelID, &ma.FileID, &ma.AssetID,
		&ma.AssetGroupID, &ma.Status, &ma.ErrorMessage, &ma.CreatedAt, &ma.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return ma, nil
}

func (s *AssetSyncStore) GetByModelAndFile(modelID, fileID string) (*ModelAsset, error) {
	ma := &ModelAsset{}
	err := s.db.QueryRow(getModelAssetByFileSQL, modelID, fileID).Scan(&ma.ID, &ma.ModelID, &ma.FileID, &ma.AssetID,
		&ma.AssetGroupID, &ma.Status, &ma.ErrorMessage, &ma.CreatedAt, &ma.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return ma, nil
}

func (s *AssetSyncStore) ListByModel(modelID string) ([]ModelAsset, error) {
	rows, err := s.db.Query(listModelAssetsSQL, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []ModelAsset
	for rows.Next() {
		var ma ModelAsset
		if err := rows.Scan(&ma.ID, &ma.ModelID, &ma.FileID, &ma.AssetID,
			&ma.AssetGroupID, &ma.Status, &ma.ErrorMessage, &ma.CreatedAt, &ma.UpdatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, ma)
	}
	return assets, nil
}

func (s *AssetSyncStore) UpdateStatus(id, status, errorMessage string) error {
	_, err := s.db.Exec(updateModelAssetStatusSQL, status, errorMessage, id)
	return err
}

func (s *AssetSyncStore) Delete(id string) error {
	_, err := s.db.Exec(deleteModelAssetSQL, id)
	return err
}

// getByFileIDsSQL returns all active sync records for the given file IDs.
// Returns a map of file_id → []ModelAsset.
func (s *AssetSyncStore) GetByFileIDs(fileIDs []string) (map[string][]ModelAsset, error) {
	result := make(map[string][]ModelAsset)
	if len(fileIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(fileIDs))
	args := make([]interface{}, len(fileIDs))
	for i, id := range fileIDs {
		args[i] = id
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(`SELECT id, model_id, file_id, asset_id, asset_group_id, status, error_message, created_at, updated_at
		FROM model_assets WHERE file_id IN (%s) AND status = 'active' ORDER BY created_at DESC`,
		strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ma ModelAsset
		if err := rows.Scan(&ma.ID, &ma.ModelID, &ma.FileID, &ma.AssetID,
			&ma.AssetGroupID, &ma.Status, &ma.ErrorMessage, &ma.CreatedAt, &ma.UpdatedAt); err != nil {
			return nil, err
		}
		result[ma.FileID] = append(result[ma.FileID], ma)
	}
	return result, nil
}
