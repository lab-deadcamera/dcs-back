package provider

import (
	"fmt"

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
	m := &Model{
		ID:                  uuid.New().String(),
		ProviderID:          req.ProviderID,
		Name:                req.Name,
		APIKey:              req.APIKey,
		URL:                 req.URL,
		Endpoint:            req.Endpoint,
		AccessKeyID:         req.AccessKeyID,
		SecretAccessKey:     req.SecretAccessKey,
		DefaultAssetGroupID: req.DefaultAssetGroupID,
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

func (s *Service) ListModels() ([]ModelWithProvider, error) {
	return s.store.ListModels()
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
	if req.Active != nil {
		updates["active"] = *req.Active
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
