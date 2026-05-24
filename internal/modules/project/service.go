package project

import (
	"fmt"

	"github.com/google/uuid"
)

// projectStore defines the storage interface needed by Service.
type projectStore interface {
	Create(p *Project) error
	GetByID(id string) (*Project, error)
	List() ([]Project, error)
	Update(id string, updates map[string]interface{}) error
	SoftDelete(id string) error

	CreateScene(sc *Scene) error
	GetSceneByID(id string) (*Scene, error)
	ListScenes(projectID string) ([]Scene, error)
	UpdateScene(id string, updates map[string]interface{}) error
	SoftDeleteScene(id string) error

	CreateTake(t *Take) error
	GetTakeByID(id string) (*Take, error)
	ListTakes(sceneID string) ([]Take, error)
	ListActiveTakes(sceneID string) ([]Take, error)
	GetActiveTakeByNumber(sceneID string, number int) (*Take, error)
	DeactivateTakesByNumber(sceneID string, number int) error
	UpdateTake(id string, updates map[string]interface{}) error
	SoftDeleteTake(id string) error
}

type Service struct {
	store projectStore
}

func NewService(store projectStore) *Service {
	return &Service{store: store}
}

// ─── Projects ───────────────────────────────────────────────────

func (s *Service) Create(req *CreateProjectRequest) (*Project, error) {
	p := &Project{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
		Active:      true,
	}
	if err := s.store.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetByID(id string) (*Project, error) {
	p, err := s.store.GetByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}
	return p, nil
}

func (s *Service) List() ([]Project, error) {
	projects, err := s.store.List()
	if err != nil {
		return nil, err
	}
	if projects == nil {
		projects = []Project{}
	}
	return projects, nil
}

func (s *Service) Update(id string, req *UpdateProjectRequest) (*Project, error) {
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
	if req.Active != nil {
		updates["active"] = *req.Active
	}

	if err := s.store.Update(id, updates); err != nil {
		return nil, err
	}
	return s.store.GetByID(id)
}

func (s *Service) SoftDelete(id string) error {
	return s.store.SoftDelete(id)
}

// ─── Scenes ─────────────────────────────────────────────────────

func (s *Service) CreateScene(projectID string, req *CreateSceneRequest) (*Scene, error) {
	// Verify project exists
	p, err := s.store.GetByID(projectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}

	sc := &Scene{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		Number:      req.Number,
		Name:        req.Name,
		Description: req.Description,
		Active:      true,
	}
	if err := s.store.CreateScene(sc); err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *Service) GetSceneByID(id string) (*Scene, error) {
	sc, err := s.store.GetSceneByID(id)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, fmt.Errorf("scene not found")
	}
	return sc, nil
}

func (s *Service) ListScenes(projectID string) ([]Scene, error) {
	// Verify project exists
	p, err := s.store.GetByID(projectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}

	scenes, err := s.store.ListScenes(projectID)
	if err != nil {
		return nil, err
	}
	if scenes == nil {
		scenes = []Scene{}
	}
	return scenes, nil
}

func (s *Service) UpdateScene(id string, req *UpdateSceneRequest) (*Scene, error) {
	updates := make(map[string]interface{})
	if req.Number != nil {
		updates["number"] = *req.Number
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}

	if err := s.store.UpdateScene(id, updates); err != nil {
		return nil, err
	}
	return s.store.GetSceneByID(id)
}

func (s *Service) SoftDeleteScene(id string) error {
	return s.store.SoftDeleteScene(id)
}

// ─── Takes ──────────────────────────────────────────────────────

func (s *Service) CreateTake(sceneID string, req *CreateTakeRequest) (*Take, error) {
	// Verify scene exists
	sc, err := s.store.GetSceneByID(sceneID)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, fmt.Errorf("scene not found")
	}

	status := req.Status
	if status == "" {
		status = "pending"
	}

	t := &Take{
		ID:      uuid.New().String(),
		SceneID: sceneID,
		Number:  req.Number,
		Status:  status,
		Active:  true,
	}
	if err := s.store.CreateTake(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) GetTakeByID(id string) (*Take, error) {
	t, err := s.store.GetTakeByID(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("take not found")
	}
	return t, nil
}

func (s *Service) ListTakes(sceneID string) ([]Take, error) {
	// Verify scene exists
	sc, err := s.store.GetSceneByID(sceneID)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, fmt.Errorf("scene not found")
	}

	takes, err := s.store.ListTakes(sceneID)
	if err != nil {
		return nil, err
	}
	if takes == nil {
		takes = []Take{}
	}
	return takes, nil
}

func (s *Service) UpdateTake(id string, req *UpdateTakeRequest) (*Take, error) {
	updates := make(map[string]interface{})
	if req.VideoURL != nil {
		updates["video_url"] = *req.VideoURL
	}
	if req.VideoLocalURL != nil {
		updates["video_local_url"] = *req.VideoLocalURL
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}

	if err := s.store.UpdateTake(id, updates); err != nil {
		return nil, err
	}
	return s.store.GetTakeByID(id)
}

func (s *Service) SoftDeleteTake(id string) error {
	return s.store.SoftDeleteTake(id)
}

// ─── Combined ───────────────────────────────────────────────────

// GetProjectWithScenes returns a project with its scenes.
func (s *Service) GetProjectWithScenes(id string) (*ProjectWithScenes, error) {
	p, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	scenes, err := s.store.ListScenes(id)
	if err != nil {
		return nil, err
	}
	if scenes == nil {
		scenes = []Scene{}
	}

	return &ProjectWithScenes{
		Project: *p,
		Scenes:  scenes,
	}, nil
}

// GetSceneWithTakes returns a scene with its takes.
func (s *Service) GetSceneWithTakes(id string) (*SceneWithTakes, error) {
	sc, err := s.GetSceneByID(id)
	if err != nil {
		return nil, err
	}

	takes, err := s.store.ListTakes(id)
	if err != nil {
		return nil, err
	}
	if takes == nil {
		takes = []Take{}
	}

	return &SceneWithTakes{
		Scene: *sc,
		Takes: takes,
	}, nil
}

// ─── Take: discard / re-generation ─────────────────────────────

// SaveGenerationRequest is the payload for associating a generated
// video URL with a take slot (scene+number).
type SaveGenerationRequest struct {
	Number   int    `json:"number"`
	VideoURL string `json:"video_url"`
}

// SaveGeneration saves a generated video URL to a take slot. If an
// active take already exists for this scene+number, it is marked as
// inactive (discarded) and a new take is created.
func (s *Service) SaveGeneration(sceneID string, req *SaveGenerationRequest) (*Take, error) {
	sc, err := s.store.GetSceneByID(sceneID)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, fmt.Errorf("scene not found")
	}

	// Deactivate any existing active take for this scene+number
	if err := s.store.DeactivateTakesByNumber(sceneID, req.Number); err != nil {
		return nil, err
	}

	t := &Take{
		ID:       uuid.New().String(),
		SceneID:  sceneID,
		Number:   req.Number,
		VideoURL: req.VideoURL,
		Status:   "completed",
		Active:   true,
	}
	if err := s.store.CreateTake(t); err != nil {
		return nil, err
	}
	return t, nil
}

// ToggleTakeActive sets the specified take as the active one and
// deactivates all other takes with the same scene+number.
func (s *Service) ToggleTakeActive(id string) (*Take, error) {
	t, err := s.store.GetTakeByID(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("take not found")
	}

	// Deactivate any active take with the same scene+number
	if err := s.store.DeactivateTakesByNumber(t.SceneID, t.Number); err != nil {
		return nil, err
	}

	// Activate this take
	active := true
	if _, err := s.UpdateTake(id, &UpdateTakeRequest{Active: &active}); err != nil {
		return nil, err
	}
	return s.store.GetTakeByID(id)
}
