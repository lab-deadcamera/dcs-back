package studio

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Store struct {
	mu           sync.RWMutex
	keysFilePath string
	presetsPath  string
	outputsDir   string
}

func NewStore(keysFilePath, presetsPath, outputsDir string) (*Store, error) {
	if err := os.MkdirAll(outputsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create outputs dir: %w", err)
	}

	return &Store{
		keysFilePath: keysFilePath,
		presetsPath:  presetsPath,
		outputsDir:   outputsDir,
	}, nil
}

func (s *Store) OutputsDir() string {
	return s.outputsDir
}

func (s *Store) LoadKeys() (*KeyStoreData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := &KeyStoreData{Keys: make([]APIKey, 0)}
	if _, err := os.Stat(s.keysFilePath); os.IsNotExist(err) {
		return data, nil
	}

	raw, err := os.ReadFile(s.keysFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys file: %w", err)
	}

	if err := json.Unmarshal(raw, data); err != nil {
		return nil, fmt.Errorf("failed to parse keys file: %w", err)
	}
	return data, nil
}

func (s *Store) SaveKeys(data *KeyStoreData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}

	return os.WriteFile(s.keysFilePath, raw, 0644)
}

func (s *Store) LoadPresets() ([]byte, error) {
	return os.ReadFile(s.presetsPath)
}
