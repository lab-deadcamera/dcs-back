package studio

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ModelAsset representa un archivo sincronizado con la galería de un modelo.
// Cada modelo puede tener su propio formato de referencia:
//
//	Modelos "gallery" (BytePlus) → ReferenceURI = "asset://" + AssetID
//	Otros modelos              → ReferenceURI = AssetURL (CDN)
type ModelAsset struct {
	ID           string    `json:"id"`
	ModelID      string    `json:"model_id"`
	FileID       string    `json:"file_id"`
	AssetID      string    `json:"asset_id"`
	AssetGroupID string    `json:"asset_group_id"`
	Status       string    `json:"status"` // "syncing", "active", "failed"
	ErrorMessage string    `json:"error_message,omitempty"`
	AssetURL     string    `json:"asset_url,omitempty"`
	AssetType    string    `json:"asset_type,omitempty"`
	ReferenceURI string    `json:"reference_uri,omitempty"` // URI lista para usar según el tipo de modelo
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ModelSyncSummary agrega los registros de model_assets por modelo,
// con conteos por estado — usado por la vista admin "Galerías Externas".
type ModelSyncSummary struct {
	ModelID  string     `json:"model_id"`
	Total    int        `json:"total"`
	Active   int        `json:"active"`
	Failed   int        `json:"failed"`
	Syncing  int        `json:"syncing"`
	LastSync *time.Time `json:"last_sync,omitempty"`
}

// ─── Queries ─────────────────────────────────────────────────────

const (
	createModelAssetSQL = `INSERT INTO model_assets (id, model_id, file_id, asset_id, asset_group_id, status, error_message, asset_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	getModelAssetSQL = `SELECT id, model_id, file_id, asset_id, asset_group_id, status, error_message, asset_url, asset_type, reference_uri, created_at, updated_at
		FROM model_assets WHERE id = $1`

	getModelAssetByFileSQL = `SELECT id, model_id, file_id, asset_id, asset_group_id, status, error_message, asset_url, asset_type, reference_uri, created_at, updated_at
		FROM model_assets WHERE model_id = $1 AND file_id = $2 ORDER BY created_at DESC LIMIT 1`

	listModelAssetsSQL = `SELECT id, model_id, file_id, asset_id, asset_group_id, status, error_message, asset_url, asset_type, reference_uri, created_at, updated_at
		FROM model_assets WHERE model_id = $1 ORDER BY created_at DESC`

	updateModelAssetStatusSQL = `UPDATE model_assets SET status = $1, error_message = $2, asset_id = $3, asset_url = $4, asset_type = $5, reference_uri = $6, updated_at = NOW()
		WHERE id = $7`

	deleteModelAssetSQL = `DELETE FROM model_assets WHERE id = $1`

	listModelSummariesSQL = `SELECT model_id, COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'active') AS active,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE status = 'syncing') AS syncing,
			MAX(updated_at) AS last_sync
		FROM model_assets GROUP BY model_id ORDER BY last_sync DESC`

	listRecentErrorsSQL = `SELECT id, model_id, file_id, asset_id, asset_group_id, status, error_message, asset_url, asset_type, reference_uri, created_at, updated_at
		FROM model_assets WHERE model_id = $1 AND file_id = $2 AND status = 'failed'
		ORDER BY created_at DESC LIMIT $3`
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
		ma.AssetGroupID, ma.Status, ma.ErrorMessage, ma.AssetType).
		Scan(&ma.CreatedAt, &ma.UpdatedAt)
}

func (s *AssetSyncStore) GetByID(id string) (*ModelAsset, error) {
	ma := &ModelAsset{}
	err := s.db.QueryRow(getModelAssetSQL, id).Scan(&ma.ID, &ma.ModelID, &ma.FileID, &ma.AssetID,
		&ma.AssetGroupID, &ma.Status, &ma.ErrorMessage, &ma.AssetURL, &ma.AssetType, &ma.ReferenceURI, &ma.CreatedAt, &ma.UpdatedAt)
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
		&ma.AssetGroupID, &ma.Status, &ma.ErrorMessage, &ma.AssetURL, &ma.AssetType, &ma.ReferenceURI, &ma.CreatedAt, &ma.UpdatedAt)
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
			&ma.AssetGroupID, &ma.Status, &ma.ErrorMessage, &ma.AssetURL, &ma.AssetType, &ma.ReferenceURI, &ma.CreatedAt, &ma.UpdatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, ma)
	}
	return assets, nil
}

// UpdateStatus persiste el resultado final de una sincronización.
// referenceURI es la URI específica del modelo (asset://id, URL directa, etc.).
func (s *AssetSyncStore) UpdateStatus(id, status, errorMessage, assetID, assetURL, assetType, referenceURI string) error {
	_, err := s.db.Exec(updateModelAssetStatusSQL, status, errorMessage, assetID, assetURL, assetType, referenceURI, id)
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

	query := fmt.Sprintf(`SELECT id, model_id, file_id, asset_id, asset_group_id, status, error_message, asset_url, asset_type, reference_uri, created_at, updated_at
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
			&ma.AssetGroupID, &ma.Status, &ma.ErrorMessage, &ma.AssetURL, &ma.AssetType, &ma.ReferenceURI, &ma.CreatedAt, &ma.UpdatedAt); err != nil {
			return nil, err
		}
		result[ma.FileID] = append(result[ma.FileID], ma)
	}
	return result, nil
}

// ListModelSummaries agrega los registros de sync por modelo — los modelos
// que tienen datos en model_assets (para la vista admin de galerías externas).
func (s *AssetSyncStore) ListModelSummaries() ([]ModelSyncSummary, error) {
	rows, err := s.db.Query(listModelSummariesSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ModelSyncSummary
	for rows.Next() {
		var m ModelSyncSummary
		if err := rows.Scan(&m.ModelID, &m.Total, &m.Active, &m.Failed, &m.Syncing, &m.LastSync); err != nil {
			return nil, err
		}
		summaries = append(summaries, m)
	}
	return summaries, nil
}

// ListRecentErrors devuelve los últimos `limit` intentos fallidos de sync
// para un archivo concreto en un modelo (status = 'failed').
func (s *AssetSyncStore) ListRecentErrors(modelID, fileID string, limit int) ([]ModelAsset, error) {
	rows, err := s.db.Query(listRecentErrorsSQL, modelID, fileID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []ModelAsset
	for rows.Next() {
		var ma ModelAsset
		if err := rows.Scan(&ma.ID, &ma.ModelID, &ma.FileID, &ma.AssetID,
			&ma.AssetGroupID, &ma.Status, &ma.ErrorMessage, &ma.AssetURL, &ma.AssetType, &ma.ReferenceURI, &ma.CreatedAt, &ma.UpdatedAt); err != nil {
			return nil, err
		}
		errors = append(errors, ma)
	}
	return errors, nil
}
