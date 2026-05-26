package provider

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"slices"

	"github.com/google/uuid"
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

// ─── Providers ──────────────────────────────────────────────────

func (s *Service) CreateProvider(req CreateProviderRequest) (*Provider, error) {
	p := &Provider{
		ID:     uuid.New().String(),
		Name:   req.Name,
		Active: true,
	}
	if err := s.store.CreateProvider(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetProviderByID(id string) (*Provider, error) {
	return s.store.GetProviderByID(id)
}

func (s *Service) ListProviders() ([]Provider, error) {
	return s.store.ListProviders()
}

func (s *Service) UpdateProvider(id string, req UpdateProviderRequest) (*Provider, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if len(updates) == 0 {
		return s.store.GetProviderByID(id)
	}
	if err := s.store.UpdateProvider(id, updates); err != nil {
		return nil, err
	}
	return s.store.GetProviderByID(id)
}

func (s *Service) SoftDeleteProvider(id string) error {
	return s.store.SoftDeleteProvider(id)
}

// ─── Models ─────────────────────────────────────────────────────

func (s *Service) CreateModel(req CreateModelRequest) (*Model, error) {
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	modelType := req.ModelType
	if modelType == "" {
		modelType = string(ModelTypeVideo)
	} else if !IsValidModelType(modelType) {
		return nil, fmt.Errorf("invalid model_type: %q (valid: video, text, audio, image)", modelType)
	}

	m := &Model{
		ID:                  uuid.New().String(),
		ProviderID:          req.ProviderID,
		Name:                req.Name,
		ModelType:           modelType,
		APIKey:              req.APIKey,
		URL:                 req.URL,
		Endpoint:            req.Endpoint,
		AccessKeyID:         req.AccessKeyID,
		SecretAccessKey:     req.SecretAccessKey,
		DefaultAssetGroupID: req.DefaultAssetGroupID,
		ProjectName:         req.ProjectName,
		ProjectNumber:       req.ProjectNumber,
		Active:              active,
	}
	if err := s.store.CreateModel(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) GetModelByID(id string) (*Model, error) {
	return s.store.GetModelByID(id)
}

func (s *Service) GetModelByName(name string) (*Model, error) {
	return s.store.GetModelByName(name)
}

func (s *Service) ListModels(modelType string) ([]ModelWithProvider, error) {
	return s.store.ListModels(modelType)
}

func (s *Service) ListModelsByProvider(providerID string) ([]Model, error) {
	p, err := s.store.GetProviderByID(providerID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("provider not found")
	}
	return s.store.ListModelsByProvider(providerID)
}

func (s *Service) UpdateModel(id string, req UpdateModelRequest) (*Model, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.APIKey != nil {
		updates["api_key"] = *req.APIKey
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Endpoint != nil {
		updates["endpoint"] = *req.Endpoint
	}
	if req.AccessKeyID != nil {
		updates["access_key_id"] = *req.AccessKeyID
	}
	if req.SecretAccessKey != nil {
		updates["secret_access_key"] = *req.SecretAccessKey
	}
	if req.DefaultAssetGroupID != nil {
		updates["default_asset_group_id"] = *req.DefaultAssetGroupID
	}
	if req.ProjectName != nil {
		updates["project_name"] = *req.ProjectName
	}
	if req.ProjectNumber != nil {
		updates["project_number"] = *req.ProjectNumber
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if req.ModelType != nil {
		if !IsValidModelType(*req.ModelType) {
			return nil, fmt.Errorf("invalid model_type: %q (valid: video, text, audio, image)", *req.ModelType)
		}
		updates["model_type"] = *req.ModelType
	}
	if len(updates) == 0 {
		return s.store.GetModelByID(id)
	}
	if err := s.store.UpdateModel(id, updates); err != nil {
		return nil, err
	}
	return s.store.GetModelByID(id)
}

func (s *Service) SoftDeleteModel(id string) error {
	return s.store.SoftDeleteModel(id)
}

func (s *Service) GetFavorite() (*Model, error) {
	return s.store.GetFavoriteModel()
}

func (s *Service) SetFavorite(id string) (*Model, error) {
	m, err := s.store.GetModelByID(id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, fmt.Errorf("model not found")
	}
	if m.Favorite {
		return m, nil
	}
	if err := s.store.UnfavoriteAll(); err != nil {
		return nil, err
	}
	if err := s.store.SetFavorite(id); err != nil {
		return nil, err
	}
	return s.store.GetModelByID(id)
}

// ─── CSV Export / Import ────────────────────────────────────────

// csvHeader defines the column order for export and import CSVs.
var csvHeader = []string{
	"provider_name", "name", "model_type",
	"api_key", "url", "endpoint",
	"access_key_id", "secret_access_key", "default_asset_group_id",
	"project_name", "project_number", "active",
}

// ExportProvidersCSV generates a CSV with all providers and their models.
func (s *Service) ExportProvidersCSV() (string, error) {
	providers, err := s.ListProvidersWithModels()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Write(csvHeader)

	for _, pwm := range providers {
		for _, m := range pwm.Models {
			active := "false"
			if m.Active {
				active = "true"
			}
			writer.Write([]string{
				pwm.Provider.Name,
				m.Name,
				m.ModelType,
				m.APIKey,
				m.URL,
				m.Endpoint,
				m.AccessKeyID,
				m.SecretAccessKey,
				m.DefaultAssetGroupID,
				m.ProjectName,
				m.ProjectNumber,
				active,
			})
		}
	}
	writer.Flush()
	return buf.String(), writer.Error()
}

// ImportProvidersCSV reads CSV rows and upserts providers + models inside a transaction.
func (s *Service) ImportProvidersCSV(r io.Reader) (*ImportResult, error) {
	reader := csv.NewReader(r)
	// Allow variable number of fields per row (some older exports may lack columns).
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must contain a header row and at least one data row")
	}

	// Locate column indices by header name so the CSV format is resilient to reordering.
	header := records[0]
	colIdx := func(name string) int {
		return slices.Index(header, name)
	}

	// Must-have columns.
	if idx := colIdx("provider_name"); idx < 0 {
		return nil, fmt.Errorf("CSV missing required column: provider_name")
	}
	if idx := colIdx("name"); idx < 0 {
		return nil, fmt.Errorf("CSV missing required column: name")
	}

	get := func(row []string, name string) string {
		idx := colIdx(name)
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return row[idx]
	}

	tx, err := s.store.DB().Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result := &ImportResult{}

	for i, row := range records[1:] {
		lineNum := i + 2
		providerName := row[colIdx("provider_name")]
		modelName := row[colIdx("name")]

		if providerName == "" || modelName == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: provider_name and name are required", lineNum))
			continue
		}

		// ── Find or create provider ──
		providerID, created, err := s.upsertProviderTx(tx, providerName)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: provider error: %v", lineNum, err))
			continue
		}
		if created {
			result.ProvidersCreated++
		}

		// ── Upsert model ──
		modelType := get(row, "model_type")
		apiKey := get(row, "api_key")
		url := get(row, "url")
		endpoint := get(row, "endpoint")

		active := true
		if a := get(row, "active"); a == "false" {
			active = false
		}

		updated, err := s.upsertModelTx(tx, providerID, modelName, modelType, apiKey, url, endpoint, active,
			get(row, "access_key_id"),
			get(row, "secret_access_key"),
			get(row, "default_asset_group_id"),
			get(row, "project_name"),
			get(row, "project_number"),
		)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: model error: %v", lineNum, err))
			continue
		}
		if updated {
			result.ModelsUpdated++
		} else {
			result.ModelsCreated++
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return result, nil
}

// upsertProviderTx looks up provider by name; creates one if it doesn't exist.
// Returns (providerID, created, error).
func (s *Service) upsertProviderTx(tx *sql.Tx, name string) (string, bool, error) {
	var id string
	err := tx.QueryRow(`SELECT id FROM providers WHERE name = $1 AND deleted_at IS NULL`, name).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		return "", false, err
	}

	id = uuid.New().String()
	_, err = tx.Exec(`INSERT INTO providers (id, name) VALUES ($1, $2)`, id, name)
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

// upsertModelTx looks up model by provider_id + name; updates it if found, creates otherwise.
// Returns (wasUpdated, error).
func (s *Service) upsertModelTx(tx *sql.Tx, providerID, name, modelType, apiKey, url, endpoint string, active bool, accessKeyID, secretAccessKey, defaultAssetGroupID, projectName, projectNumber string) (bool, error) {
	var existingID string
	err := tx.QueryRow(`SELECT id FROM models WHERE provider_id = $1 AND name = $2 AND deleted_at IS NULL`, providerID, name).Scan(&existingID)
	if err == nil {
		// Update existing model — keep its api_key if the CSV value is empty (sensitive field).
		if apiKey == "" {
			apiKey = "pending"
		}
		_, err = tx.Exec(`
			UPDATE models SET
				model_type = $1, api_key = $2, url = $3, endpoint = $4,
				access_key_id = $5, secret_access_key = $6,
				default_asset_group_id = $7, project_name = $8, project_number = $9,
				active = $10, updated_at = NOW()
			WHERE id = $11`, modelType, apiKey, url, endpoint,
			accessKeyID, secretAccessKey, defaultAssetGroupID, projectName, projectNumber,
			active, existingID)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}

	// Create new model.
	if apiKey == "" {
		apiKey = "pending"
	}
	id := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO models (id, provider_id, name, model_type, api_key, url, endpoint,
		                    access_key_id, secret_access_key, default_asset_group_id,
		                    project_name, project_number, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		id, providerID, name, modelType, apiKey, url, endpoint,
		accessKeyID, secretAccessKey, defaultAssetGroupID, projectName, projectNumber,
		active)
	if err != nil {
		return false, err
	}
	return false, nil
}

// ─── Compound ───────────────────────────────────────────────────

func (s *Service) GetProviderWithModels(id string) (*ProviderWithModels, error) {
	p, err := s.store.GetProviderByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	models, err := s.store.ListModelsByProvider(id)
	if err != nil {
		return nil, err
	}
	if models == nil {
		models = []Model{}
	}
	return &ProviderWithModels{Provider: *p, Models: models}, nil
}

func (s *Service) ListProvidersWithModels() ([]ProviderWithModels, error) {
	providers, err := s.store.ListProviders()
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return []ProviderWithModels{}, nil
	}

	ids := make([]string, len(providers))
	for i, p := range providers {
		ids[i] = p.ID
	}

	modelsMap, err := s.store.ListModelsForProviders(ids)
	if err != nil {
		return nil, err
	}

	result := make([]ProviderWithModels, len(providers))
	for i, p := range providers {
		models := modelsMap[p.ID]
		if models == nil {
			models = []Model{}
		}
		result[i] = ProviderWithModels{Provider: p, Models: models}
	}
	return result, nil
}
