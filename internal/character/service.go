package character

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

func (s *Service) Create(req CreateCharacterRequest) (*Character, error) {
	ch := &Character{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
	if ch.Metadata == "" {
		ch.Metadata = "{}"
	}
	if err := s.store.Create(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *Service) GetByID(id string) (*Character, error) {
	return s.store.GetByID(id)
}

func (s *Service) GetByIDWithFiles(id string) (*CharacterWithFiles, error) {
	ch, err := s.store.GetByID(id)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, nil
	}
	files, err := s.store.ListFiles(id)
	if err != nil {
		return nil, err
	}
	if files == nil {
		files = []FileRef{}
	}
	return &CharacterWithFiles{Character: *ch, Files: files}, nil
}

func (s *Service) List() ([]Character, error) {
	return s.store.List()
}

func (s *Service) Update(id string, req UpdateCharacterRequest) (*Character, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Metadata != nil {
		updates["metadata"] = *req.Metadata
	}
	if len(updates) == 0 {
		return s.store.GetByID(id)
	}
	if err := s.store.Update(id, updates); err != nil {
		return nil, err
	}
	return s.store.GetByID(id)
}

func (s *Service) SoftDelete(id string) error {
	return s.store.SoftDelete(id)
}

func (s *Service) AddFile(characterID, fileID, role string) error {
	ch, err := s.store.GetByID(characterID)
	if err != nil {
		return err
	}
	if ch == nil {
		return fmt.Errorf("character not found")
	}
	return s.store.AddFile(characterID, fileID, role)
}

func (s *Service) RemoveFile(characterID, fileID string) error {
	return s.store.RemoveFile(characterID, fileID)
}

func (s *Service) ListFiles(characterID string) ([]FileRef, error) {
	return s.store.ListFiles(characterID)
}
