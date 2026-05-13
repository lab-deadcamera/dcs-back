package character

import (
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	store   *Store
	baseURL string
}

func NewService(store *Store, baseURL string) *Service {
	return &Service{store: store, baseURL: baseURL}
}

func (s *Service) enrichFileURLs(files []CharacterFile) {
	for i := range files {
		files[i].URL = s.baseURL + "/api/v1/files/" + files[i].FileID + "/serve"
	}
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
		files = []CharacterFile{}
	}
	s.enrichFileURLs(files)
	return &CharacterWithFiles{Character: *ch, Files: files}, nil
}

func (s *Service) List() ([]CharacterWithFiles, error) {
	chars, err := s.store.List()
	if err != nil {
		return nil, err
	}
	if len(chars) == 0 {
		return []CharacterWithFiles{}, nil
	}

	ids := make([]string, len(chars))
	for i, ch := range chars {
		ids[i] = ch.ID
	}

	filesMap, err := s.store.ListFilesForCharacters(ids)
	if err != nil {
		return nil, err
	}

	result := make([]CharacterWithFiles, len(chars))
	for i, ch := range chars {
		files := filesMap[ch.ID]
		if files == nil {
			files = []CharacterFile{}
		}
		s.enrichFileURLs(files)
		result[i] = CharacterWithFiles{Character: ch, Files: files}
	}
	return result, nil
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

func (s *Service) ListFiles(characterID string) ([]CharacterFile, error) {
	files, err := s.store.ListFiles(characterID)
	if err != nil {
		return nil, err
	}
	if files == nil {
		files = []CharacterFile{}
	}
	s.enrichFileURLs(files)
	return files, nil
}
