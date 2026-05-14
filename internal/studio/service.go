package studio

import (
	"fmt"
	"sync"

	"dcs-back-v0/internal/provider"
)

type Service struct {
	providerStore *provider.Store
	outputsDir    string
	handlers      []ModelHandler
	tasks         map[string]*TaskRecord
	mu            sync.RWMutex
}

func NewService(providerStore *provider.Store, outputsDir string) *Service {
	return &Service{
		providerStore: providerStore,
		outputsDir:    outputsDir,
		handlers:      []ModelHandler{},
		tasks:         make(map[string]*TaskRecord),
	}
}

func (s *Service) RegisterHandler(h ModelHandler) {
	s.handlers = append(s.handlers, h)
}

func (s *Service) pickHandler(modelName string) ModelHandler {
	for _, h := range s.handlers {
		if h.Matches(modelName) {
			return h
		}
	}
	return nil
}

func (s *Service) Generate(sel *Selection) (*GenerateResponse, error) {
	m, err := s.providerStore.GetModelByID(sel.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("model not found")
	}

	handler := s.pickHandler(m.Name)
	if handler == nil {
		return nil, fmt.Errorf("no handler available for model: %s", m.Name)
	}

	resp, err := handler.Generate(sel, m.APIKey, m.URL, m.Endpoint)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.tasks[resp.TaskID] = &TaskRecord{
		TaskID:    resp.TaskID,
		ModelID:   m.ID,
		ModelName: m.Name,
		Status:    "running",
	}
	s.mu.Unlock()

	return resp, nil
}

func (s *Service) GetStatus(taskID string) (*StatusResult, error) {
	s.mu.RLock()
	record, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if !ok {
		// Try to find model ID — we need it for the handler lookup
		// For now, iterate all tasks
		return nil, fmt.Errorf("unknown task: %s", taskID)
	}

	m, err := s.providerStore.GetModelByID(record.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("model for task %s not found", taskID)
	}

	handler := s.pickHandler(m.Name)
	if handler == nil {
		return nil, fmt.Errorf("no handler available for model: %s", m.Name)
	}

	result, err := handler.GetStatus(taskID, m.APIKey, m.URL, m.Endpoint)
	if err != nil {
		return nil, err
	}

	if result.Status == "succeeded" || result.Status == "failed" {
		s.mu.Lock()
		record.Status = result.Status
		record.Result = result
		s.mu.Unlock()
	}

	return result, nil
}

func (s *Service) CancelTask(taskID string) error {
	s.mu.RLock()
	record, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown task: %s", taskID)
	}

	m, err := s.providerStore.GetModelByID(record.ModelID)
	if err != nil {
		return err
	}
	if m == nil {
		return fmt.Errorf("model for task %s not found", taskID)
	}

	handler := s.pickHandler(m.Name)
	if handler == nil {
		return fmt.Errorf("no handler available for model: %s", m.Name)
	}

	return handler.CancelTask(taskID, m.APIKey, m.URL, m.Endpoint)
}
