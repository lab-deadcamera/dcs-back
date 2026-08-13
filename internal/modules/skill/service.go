package skill

import (
	"github.com/google/uuid"
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(req *CreateSkillRequest) (*Skill, error) {
	skill := &Skill{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
	}
	if err := s.store.Create(skill); err != nil {
		return nil, err
	}
	return skill, nil
}

func (s *Service) GetByID(id string) (*Skill, error) {
	return s.store.GetByID(id)
}

func (s *Service) List() ([]Skill, error) {
	return s.store.List()
}

func (s *Service) Update(id string, req *UpdateSkillRequest) error {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.SystemPrompt != nil {
		updates["system_prompt"] = *req.SystemPrompt
	}
	return s.store.Update(id, updates)
}

func (s *Service) Delete(id string) error {
	return s.store.SoftDelete(id)
}
