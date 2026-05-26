package project

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TaskLookup resolves a completed generation task to its local video URL.
// Returns empty string if the task is not found or not yet completed.
type TaskLookup func(taskID string) string

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

	GetScenePresets(sceneID string) ([]ScenePresetAssignment, error)
	GetSceneCharacters(sceneID string) ([]SceneCharacterAssignment, error)
	GetSceneAssets(sceneID string) ([]SceneAssetAssignment, error)
	AssignPresetToScene(sceneID, presetID string) (string, error)
	AssignCharacterToScene(sceneID, characterID string) (string, error)
	AssignAssetToScene(sceneID, fileID string) (string, error)
	RemoveScenePreset(assignmentID string) error
	RemoveSceneCharacter(assignmentID string) error
	RemoveSceneAsset(assignmentID string) error
}

type Service struct {
	store      projectStore
	taskLookup TaskLookup
	outputsDir string
}

func NewService(store projectStore) *Service {
	return &Service{store: store}
}

// SetOutputsDir sets the filesystem path where generated videos are stored.
func (s *Service) SetOutputsDir(dir string) {
	s.outputsDir = dir
}

// SetTaskLookup injects an optional function to resolve local URLs
// from completed generation tasks.
func (s *Service) SetTaskLookup(lookup TaskLookup) {
	s.taskLookup = lookup
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
	Number        int    `json:"number"`
	VideoURL      string `json:"video_url"`
	VideoLocalURL string `json:"video_local_url"`
	TaskID        string `json:"task_id"`
}

// SaveGeneration saves a generated video URL to a take slot. If an
// active take already exists for this scene+number, it is marked as
// inactive (discarded) and a new take is created.
// If task_id is provided and a TaskLookup is configured, the local
// video URL is resolved automatically from the completed task.
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

	localURL := req.VideoLocalURL
	if localURL == "" && req.TaskID != "" && s.taskLookup != nil {
		localURL = s.taskLookup(req.TaskID)
	}

	videoURL := req.VideoURL
	if localURL != "" {
		videoURL = localURL
	}

	t := &Take{
		ID:            uuid.New().String(),
		SceneID:       sceneID,
		Number:        req.Number,
		VideoURL:      videoURL,
		VideoLocalURL: localURL,
		Status:        "completed",
		Active:        true,
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

// ─── Scene Assignments ──────────────────────────────────────────

func (s *Service) GetSceneAssignments(sceneID string) (*SceneAssignments, error) {
	presets, err := s.store.GetScenePresets(sceneID)
	if err != nil {
		return nil, err
	}
	characters, err := s.store.GetSceneCharacters(sceneID)
	if err != nil {
		return nil, err
	}
	assets, err := s.store.GetSceneAssets(sceneID)
	if err != nil {
		return nil, err
	}
	return &SceneAssignments{
		Presets:    presets,
		Characters: characters,
		Assets:     assets,
	}, nil
}

func (s *Service) AssignPresetToScene(sceneID, presetID string) (string, error) {
	return s.store.AssignPresetToScene(sceneID, presetID)
}

func (s *Service) AssignCharacterToScene(sceneID, characterID string) (string, error) {
	return s.store.AssignCharacterToScene(sceneID, characterID)
}

func (s *Service) AssignAssetToScene(sceneID, fileID string) (string, error) {
	return s.store.AssignAssetToScene(sceneID, fileID)
}

func (s *Service) RemoveScenePreset(assignmentID string) error {
	return s.store.RemoveScenePreset(assignmentID)
}

func (s *Service) RemoveSceneCharacter(assignmentID string) error {
	return s.store.RemoveSceneCharacter(assignmentID)
}

func (s *Service) RemoveSceneAsset(assignmentID string) error {
	return s.store.RemoveSceneAsset(assignmentID)
}

// DownloadTakeVideo downloads the external video for a take and saves it
// locally under outputsDir/{ProjectName}/{SceneCode}/T{take}/{user}_{datetime}.mp4.
// Returns the updated take with video_local_url populated.
func (s *Service) DownloadTakeVideo(takeID, username string) (*Take, error) {
	t, err := s.store.GetTakeByID(takeID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("take not found")
	}
	if t.VideoURL == "" {
		return nil, fmt.Errorf("take has no video URL")
	}
	if t.VideoLocalURL != "" {
		return t, nil
	}

	sc, err := s.store.GetSceneByID(t.SceneID)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, fmt.Errorf("scene not found for take")
	}

	proj, err := s.store.GetByID(sc.ProjectID)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, fmt.Errorf("project not found for take")
	}

	now := time.Now()
	safe := func(s string) string {
		r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
		return r.Replace(s)
	}

	sceneCode := fmt.Sprintf("SC%02d", sc.Number)
	userPart := username
	if userPart == "" {
		userPart = "unknown"
	}
	ts := now.Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_T%d_%s_%s.mp4", safe(proj.Name), sceneCode, t.Number, safe(userPart), ts)
	localPath := filepath.Join(s.outputsDir, filename)

	resp, err := http.Get(t.VideoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read video: %w", err)
	}

	if err := os.WriteFile(localPath, body, 0644); err != nil {
		return nil, fmt.Errorf("failed to write video: %w", err)
	}

	localURL := "/outputs/" + filename

	if err := s.store.UpdateTake(takeID, map[string]interface{}{
		"video_local_url": localURL,
	}); err != nil {
		return nil, err
	}

	t.VideoLocalURL = localURL
	return t, nil
}
