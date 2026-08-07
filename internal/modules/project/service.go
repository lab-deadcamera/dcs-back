package project

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dcs-back-v0/config"

	"github.com/google/uuid"
)

// TaskLookup resolves a completed generation task to its local video URL.
// Returns empty string if the task is not found or not yet completed.
type TaskLookup func(taskID string) string

// Sentinel errors for delete protections (scenes/shots with existing takes).
var (
	ErrSceneHasTakes = errors.New("scene has existing takes")
	ErrShotHasTakes  = errors.New("shot has existing takes")
)

// projectStore defines the storage interface needed by Service.
type projectStore interface {
	Create(p *Project) error
	GetByID(id string) (*Project, error)
	List() ([]Project, error)
	ListAll() ([]Project, error)
	Update(id string, updates map[string]interface{}) error
	SoftDelete(id string) error

	CreateChapter(c *Chapter) error
	GetChapterByID(id string) (*Chapter, error)
	ListChapters(projectID string) ([]Chapter, error)
	UpdateChapter(id string, updates map[string]interface{}) error
	SoftDeleteChapter(id string) error

	CreateScene(sc *Scene) error
	GetSceneByID(id string) (*Scene, error)
	ListScenes(chapterID string) ([]Scene, error)
	UpdateScene(id string, updates map[string]interface{}) error
	SoftDeleteScene(id string) error

	CreateShot(sh *Shot) error
	GetShotByID(id string) (*Shot, error)
	ListShots(sceneID string) ([]Shot, error)
	UpdateShot(id string, updates map[string]interface{}) error
	SoftDeleteShot(id string) error

	CreateTake(t *Take) error
	GetTakeByID(id string) (*Take, error)
	ListTakes(shotID string) ([]Take, error)
	ListActiveTakes(shotID string) ([]Take, error)
	GetActiveTakeByNumber(shotID string, number int) (*Take, error)
	GetPendingTakeByNumber(shotID string, number int) (*Take, error)
	DeactivateTakesByNumber(shotID string, number int) error
	DeactivateFinalsByNumber(shotID string, number int) error
	GetTakeByVideoURL(shotID string, videoURL string) (*Take, error)
	UpdateTake(id string, updates map[string]interface{}) error
	SoftDeleteTake(id string) error

	GetScenePresets(sceneID string) ([]ScenePresetAssignment, error)
	GetSceneCharacters(sceneID string) ([]SceneCharacterAssignment, error)
	GetSceneAssets(sceneID string) ([]SceneAssetAssignment, error)
	AssignPresetToScene(sceneID, presetID string) (string, error)
	AssignCharacterToScene(sceneID, characterID, slot string) (string, error)
	AssignAssetToScene(sceneID, fileID string) (string, error)
	RemoveScenePreset(assignmentID string) error
	RemoveSceneCharacter(assignmentID string) error
	RemoveSceneAsset(assignmentID string) error

	AssignCharacterToShot(shotID, characterID, slot string) (string, error)
	AssignAssetToShot(shotID, fileID, slot string) (string, error)
	AssignPresetToShot(shotID, presetID string) (string, error)
	RemoveShotCharacter(assignmentID string) error
	RemoveShotAsset(assignmentID string) error
	RemoveShotPreset(assignmentID string) error
	UpdateShotModel(shotID, modelID string) error
		GetChapterCharacters(chapterID string) ([]ChapterCharacterAssignment, error)
		GetChapterAssets(chapterID string) ([]ChapterAssetAssignment, error)
		GetChapterPresets(chapterID string) ([]ChapterPresetAssignment, error)
		AssignCharacterToChapter(chapterID, characterID, slot string) (string, error)
		AssignAssetToChapter(chapterID, fileID, slot string) (string, error)
		AssignPresetToChapter(chapterID, presetID string) (string, error)
		RemoveChapterCharacter(assignmentID string) error
		RemoveChapterAsset(assignmentID string) error
		RemoveChapterPreset(assignmentID string) error
}

type Service struct {
	store      projectStore
	taskLookup TaskLookup
}

func NewService(store projectStore) *Service {
	return &Service{store: store}
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

func (s *Service) ListAll() ([]Project, error) {
	projects, err := s.store.ListAll()
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

// ─── Chapters ───────────────────────────────────────────────────

func (s *Service) CreateChapter(projectID string, req *CreateChapterRequest) (*Chapter, error) {
	p, err := s.store.GetByID(projectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}

	c := &Chapter{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		Number:      req.Number,
		Name:        req.Name,
		Description: req.Description,
		Active:      true,
	}
	if err := s.store.CreateChapter(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) GetChapterByID(id string) (*Chapter, error) {
	c, err := s.store.GetChapterByID(id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("chapter not found")
	}
	return c, nil
}

func (s *Service) ListChapters(projectID string) ([]Chapter, error) {
	p, err := s.store.GetByID(projectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}

	chapters, err := s.store.ListChapters(projectID)
	if err != nil {
		return nil, err
	}
	if chapters == nil {
		chapters = []Chapter{}
	}
	return chapters, nil
}

func (s *Service) UpdateChapter(id string, req *UpdateChapterRequest) (*Chapter, error) {
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

	if err := s.store.UpdateChapter(id, updates); err != nil {
		return nil, err
	}
	return s.store.GetChapterByID(id)
}

func (s *Service) SoftDeleteChapter(id string) error {
	return s.store.SoftDeleteChapter(id)
}

// ─── Scenes ─────────────────────────────────────────────────────

func (s *Service) CreateScene(chapterID string, req *CreateSceneRequest) (*Scene, error) {
	// Verify chapter exists
	c, err := s.store.GetChapterByID(chapterID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("chapter not found")
	}

	sc := &Scene{
		ID:          uuid.New().String(),
		ProjectID:   c.ProjectID,
		ChapterID:   chapterID,
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

func (s *Service) ListScenes(chapterID string) ([]Scene, error) {
	c, err := s.store.GetChapterByID(chapterID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("chapter not found")
	}

	scenes, err := s.store.ListScenes(chapterID)
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
	sc, err := s.store.GetSceneByID(id)
	if err != nil {
		return err
	}
	if sc == nil {
		return fmt.Errorf("scene not found")
	}
	if sc.TakeCount > 0 {
		return ErrSceneHasTakes
	}
	return s.store.SoftDeleteScene(id)
}

// ─── Shots ──────────────────────────────────────────────────────

func (s *Service) CreateShot(sceneID string, req *CreateShotRequest) (*Shot, error) {
	// Verify scene exists
	sc, err := s.store.GetSceneByID(sceneID)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, fmt.Errorf("scene not found")
	}

	sh := &Shot{
		ID:          uuid.New().String(),
		SceneID:     sceneID,
		Number:      req.Number,
		Name:        req.Name,
		Description: req.Description,
		Active:      true,
		AspectRatio:     req.AspectRatio,
		DurationSeconds: req.DurationSeconds,
	}
	if err := s.store.CreateShot(sh); err != nil {
		return nil, err
	}
	return sh, nil
}

func (s *Service) GetShotByID(id string) (*Shot, error) {
	sh, err := s.store.GetShotByID(id)
	if err != nil {
		return nil, err
	}
	if sh == nil {
		return nil, fmt.Errorf("shot not found")
	}
	return sh, nil
}

func (s *Service) ListShots(sceneID string) ([]Shot, error) {
	sc, err := s.store.GetSceneByID(sceneID)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, fmt.Errorf("scene not found")
	}

	shots, err := s.store.ListShots(sceneID)
	if err != nil {
		return nil, err
	}
	if shots == nil {
		shots = []Shot{}
	}
	return shots, nil
}

func (s *Service) UpdateShot(id string, req *UpdateShotRequest) (*Shot, error) {
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
	if req.AspectRatio != nil {
		updates["aspect_ratio"] = *req.AspectRatio
	}
	if req.DurationSeconds != nil {
		updates["duration_seconds"] = *req.DurationSeconds
	}

	if err := s.store.UpdateShot(id, updates); err != nil {
		return nil, err
	}
	return s.store.GetShotByID(id)
}

func (s *Service) SoftDeleteShot(id string) error {
	sh, err := s.store.GetShotByID(id)
	if err != nil {
		return err
	}
	if sh == nil {
		return fmt.Errorf("shot not found")
	}
	if sh.TakeCount > 0 {
		return ErrShotHasTakes
	}
	return s.store.SoftDeleteShot(id)
}

// ─── Takes ──────────────────────────────────────────────────────

func (s *Service) CreateTake(shotID string, req *CreateTakeRequest) (*Take, error) {
	// Verify shot exists
	sh, err := s.store.GetShotByID(shotID)
	if err != nil {
		return nil, err
	}
	if sh == nil {
		return nil, fmt.Errorf("shot not found")
	}

	status := req.Status
	if status == "" {
		status = "pending"
	}

	t := &Take{
		ID:      uuid.New().String(),
		SceneID: sh.SceneID,
		ShotID:  shotID,
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

func (s *Service) ListTakes(shotID string) ([]Take, error) {
	sh, err := s.store.GetShotByID(shotID)
	if err != nil {
		return nil, err
	}
	if sh == nil {
		return nil, fmt.Errorf("shot not found")
	}

	takes, err := s.store.ListTakes(shotID)
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
	if req.Final != nil && *req.Final {
		t, err := s.store.GetTakeByID(id)
		if err != nil {
			return nil, err
		}
		if t == nil {
			return nil, fmt.Errorf("take not found")
		}
		if err := s.store.DeactivateFinalsByNumber(t.ShotID, t.Number); err != nil {
			return nil, err
		}
		updates["final"] = true
		updates["finalized_at"] = time.Now()
	}
	if req.TaskID != nil {
		updates["task_id"] = *req.TaskID
	}
	if req.Rating != nil {
		updates["rating"] = *req.Rating
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

// GetProjectWithChapters returns a project with its chapters.
func (s *Service) GetProjectWithChapters(id string) (*ProjectWithChapters, error) {
	p, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	chapters, err := s.store.ListChapters(id)
	if err != nil {
		return nil, err
	}
	if chapters == nil {
		chapters = []Chapter{}
	}

	return &ProjectWithChapters{
		Project:  *p,
		Chapters: make([]ChapterWithScenes, len(chapters)),
	}, nil
}

// GetChapterWithScenes returns a chapter with its scenes.
func (s *Service) GetChapterWithScenes(id string) (*ChapterWithScenes, error) {
	c, err := s.GetChapterByID(id)
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

	sceneWithShots := make([]SceneWithShots, len(scenes))
	for i, sc := range scenes {
		sceneWithShots[i] = SceneWithShots{
			Scene: sc,
			Shots: []ShotWithTakes{},
		}
	}

	return &ChapterWithScenes{
		Chapter: *c,
		Scenes:  sceneWithShots,
	}, nil
}

// GetSceneWithShots returns a scene with its shots.
func (s *Service) GetSceneWithShots(id string) (*SceneWithShots, error) {
	sc, err := s.GetSceneByID(id)
	if err != nil {
		return nil, err
	}

	shots, err := s.store.ListShots(id)
	if err != nil {
		return nil, err
	}
	if shots == nil {
		shots = []Shot{}
	}

	shotWithTakes := make([]ShotWithTakes, len(shots))
	for i, sh := range shots {
		shotWithTakes[i] = ShotWithTakes{
			Shot:  sh,
			Takes: []Take{},
		}
	}

	return &SceneWithShots{
		Scene: *sc,
		Shots: shotWithTakes,
	}, nil
}

// GetShotWithTakes returns a shot with its takes.
func (s *Service) GetShotWithTakes(id string) (*ShotWithTakes, error) {
	sh, err := s.GetShotByID(id)
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

	return &ShotWithTakes{
		Shot:  *sh,
		Takes: takes,
	}, nil
}

// Legacy GetProjectWithScenes for backward compatibility — returns project with
// flat scenes list (no chapters, no shots nesting).
func (s *Service) GetProjectWithScenes(id string) (*ProjectWithScenes, error) {
	p, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// We need to flatten all scenes across all chapters
	chapters, err := s.store.ListChapters(id)
	if err != nil {
		return nil, err
	}

	var allScenes []Scene
	for _, c := range chapters {
		scenes, err := s.store.ListScenes(c.ID)
		if err != nil {
			return nil, err
		}
		allScenes = append(allScenes, scenes...)
	}
	if allScenes == nil {
		allScenes = []Scene{}
	}

	return &ProjectWithScenes{
		Project: *p,
		Scenes:  allScenes,
	}, nil
}

// Legacy GetSceneWithTakes for backward compatibility.
func (s *Service) GetSceneWithTakes(id string) (*SceneWithTakes, error) {
	sc, err := s.GetSceneByID(id)
	if err != nil {
		return nil, err
	}

	// Flatten all takes across all shots in this scene
	shots, err := s.store.ListShots(id)
	if err != nil {
		return nil, err
	}

	var allTakes []Take
	for _, sh := range shots {
		takes, err := s.store.ListTakes(sh.ID)
		if err != nil {
			return nil, err
		}
		allTakes = append(allTakes, takes...)
	}
	if allTakes == nil {
		allTakes = []Take{}
	}

	return &SceneWithTakes{
		Scene: *sc,
		Takes: allTakes,
	}, nil
}

// ─── Take: discard / re-generation ─────────────────────────────

// SaveGenerationRequest is the payload for associating a generated
// video URL with a take slot (shot+number).
type SaveGenerationRequest struct {
	Number        int    `json:"number"`
	VideoURL      string `json:"video_url"`
	VideoLocalURL string `json:"video_local_url"`
	TaskID        string `json:"task_id"`
}

// SaveGeneration saves a generated video URL to a take.
// Rules:
//  1. If a pending take exists for this shot+number, it is updated with
//     the completed video URLs and status.
//  2. Otherwise, a new take is created. Multiple completed takes with
//     the same shot+number are allowed (each with a different video_url).
//
// If task_id is provided and a TaskLookup is configured, the local
// video URL is resolved automatically from the completed task.
func (s *Service) SaveGeneration(shotID string, req *SaveGenerationRequest) (*Take, error) {
	sh, err := s.store.GetShotByID(shotID)
	if err != nil {
		return nil, err
	}
	if sh == nil {
		return nil, fmt.Errorf("shot not found")
	}

	localURL := req.VideoLocalURL
	if localURL == "" && req.TaskID != "" && s.taskLookup != nil {
		localURL = s.taskLookup(req.TaskID)
	}

	videoURL := req.VideoURL
	if videoURL == "" && localURL != "" {
		videoURL = localURL
	}

	// Rule 1: If an active take exists for this shot+number, update it
	take, err := s.store.GetActiveTakeByNumber(shotID, req.Number)
	if err == nil && take != nil {
		updates := map[string]interface{}{
			"video_url":       videoURL,
			"video_local_url": localURL,
			"status":          "completed",
		}
		if req.TaskID != "" {
			updates["task_id"] = req.TaskID
		}
		if err := s.store.UpdateTake(take.ID, updates); err != nil {
			return nil, fmt.Errorf("failed to update take %s: %w", take.ID, err)
		}
		return s.store.GetTakeByID(take.ID)
	}

	// Rule 2: Create a new take (duplicate shot+number is allowed)
	t := &Take{
		ID:            uuid.New().String(),
		SceneID:       sh.SceneID,
		ShotID:        shotID,
		Number:        req.Number,
		VideoURL:      videoURL,
		VideoLocalURL: localURL,
		Status:        "completed",
		Active:        true,
		TaskID:        req.TaskID,
	}

	if err := s.store.CreateTake(t); err != nil {
		return nil, err
	}
	return t, nil
}

// ToggleTakeActive sets the specified take as the active one and
// deactivates all other takes with the same shot+number.
func (s *Service) ToggleTakeActive(id string) (*Take, error) {
	t, err := s.store.GetTakeByID(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("take not found")
	}

	// Deactivate any active take with the same shot+number
	if err := s.store.DeactivateTakesByNumber(t.ShotID, t.Number); err != nil {
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

func (s *Service) AssignCharacterToScene(sceneID, characterID, slot string) (string, error) {
	return s.store.AssignCharacterToScene(sceneID, characterID, slot)
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

// ─── Shot Resources ────────────────────────────────────────────


func (s *Service) AssignCharacterToShot(shotID, characterID, slot string) (string, error) {
	return s.store.AssignCharacterToShot(shotID, characterID, slot)
}

func (s *Service) AssignAssetToShot(shotID, fileID, slot string) (string, error) {
	return s.store.AssignAssetToShot(shotID, fileID, slot)
}

func (s *Service) AssignPresetToShot(shotID, presetID string) (string, error) {
	return s.store.AssignPresetToShot(shotID, presetID)
}

func (s *Service) RemoveShotCharacter(assignmentID string) error {
	return s.store.RemoveShotCharacter(assignmentID)
}

func (s *Service) RemoveShotAsset(assignmentID string) error {
	return s.store.RemoveShotAsset(assignmentID)
}

func (s *Service) RemoveShotPreset(assignmentID string) error {
	return s.store.RemoveShotPreset(assignmentID)
}

func (s *Service) UpdateShotModel(shotID, modelID string) error {
	return s.store.UpdateShotModel(shotID, modelID)
}

// DownloadTakeVideo downloads the external video for a take and saves it
// locally under outputsDir/{ProjectName}_{SceneCode}_S{shot}_T{take}_{user}_{datetime}.mp4.
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

	// Resolve shot -> scene -> project for naming
	sh, err := s.store.GetShotByID(t.ShotID)
	if err != nil {
		return nil, err
	}
	if sh == nil {
		return nil, fmt.Errorf("shot not found for take")
	}

	sc, err := s.store.GetSceneByID(sh.SceneID)
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
	shotCode := fmt.Sprintf("S%02d", sh.Number)
	userPart := username
	if userPart == "" {
		userPart = "unknown"
	}
	ts := now.Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_%s_T%d_%s_%s.mp4", safe(proj.Name), sceneCode, shotCode, t.Number, safe(userPart), ts)
	outputsDir := filepath.Join(".", config.OutPutUrl())
	localPath := filepath.Join(outputsDir, filename)

	if err := os.MkdirAll(outputsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create outputs dir: %w", err)
	}

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

	localURL := config.OutPutUrl() + "/" + filename

	if err := s.store.UpdateTake(takeID, map[string]interface{}{
		"video_url":       localURL,
		"video_local_url": localURL,
	}); err != nil {
		return nil, err
	}

	t.VideoURL = localURL
	t.VideoLocalURL = localURL
	return t, nil
}

// ─── Chapter Assignment Service Methods ──────────────────────────

func (s *Service) GetChapterAssignments(chapterID string) (*ChapterAssignments, error) {
	characters, err := s.store.GetChapterCharacters(chapterID)
	if err != nil {
		return nil, err
	}
	assets, err := s.store.GetChapterAssets(chapterID)
	if err != nil {
		return nil, err
	}
	presets, err := s.store.GetChapterPresets(chapterID)
	if err != nil {
		return nil, err
	}
	return &ChapterAssignments{
		Characters: characters,
		Assets:     assets,
		Presets:    presets,
	}, nil
}

func (s *Service) AssignCharacterToChapter(chapterID, characterID, slot string) (string, error) {
	return s.store.AssignCharacterToChapter(chapterID, characterID, slot)
}

func (s *Service) AssignAssetToChapter(chapterID, fileID, slot string) (string, error) {
	return s.store.AssignAssetToChapter(chapterID, fileID, slot)
}

func (s *Service) AssignPresetToChapter(chapterID, presetID string) (string, error) {
	return s.store.AssignPresetToChapter(chapterID, presetID)
}

func (s *Service) RemoveChapterCharacter(assignmentID string) error {
	return s.store.RemoveChapterCharacter(assignmentID)
}

func (s *Service) RemoveChapterAsset(assignmentID string) error {
	return s.store.RemoveChapterAsset(assignmentID)
}

func (s *Service) RemoveChapterPreset(assignmentID string) error {
	return s.store.RemoveChapterPreset(assignmentID)
}
