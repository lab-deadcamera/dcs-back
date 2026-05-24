package preset

import "github.com/google/uuid"

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

// ─── Groups ─────────────────────────────────────────────────────

func (s *Service) ListGroups(includeInactive bool) ([]Group, error) {
	return s.store.ListGroups(includeInactive)
}

func (s *Service) CreateGroup(req *CreateGroupRequest) (*Group, error) {
	g := &Group{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Active:      true,
	}
	if err := s.store.CreateGroup(g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Service) UpdateGroup(id string, req *UpdateGroupRequest) (*Group, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if err := s.store.UpdateGroup(id, updates); err != nil {
		return nil, err
	}
	// Return updated group — re-fetch is overkill, caller can rely on response
	return &Group{ID: id}, nil
}

// ─── Presets ────────────────────────────────────────────────────

func (s *Service) ListPresets(groupID string, includeInactive bool) ([]Preset, error) {
	return s.store.ListPresets(groupID, includeInactive)
}

func (s *Service) GetPresetByID(id string) (*Preset, error) {
	return s.store.GetPresetByID(id)
}

func (s *Service) CreatePreset(req *CreatePresetRequest) (*Preset, error) {
	p := &Preset{
		ID:      uuid.New().String(),
		GroupID: req.GroupID,
		Code:    req.Code,
		Label:   req.Label,
		Prompt:  req.Prompt,
		Active:  true,
	}
	if err := s.store.CreatePreset(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) UpdatePreset(id string, req *UpdatePresetRequest) (*Preset, error) {
	updates := make(map[string]interface{})
	if req.Code != nil {
		updates["code"] = *req.Code
	}
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.Prompt != nil {
		updates["prompt"] = *req.Prompt
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if err := s.store.UpdatePreset(id, updates); err != nil {
		return nil, err
	}
	return &Preset{ID: id}, nil
}

func (s *Service) SoftDeletePreset(id string) error {
	return s.store.SoftDeletePreset(id)
}
