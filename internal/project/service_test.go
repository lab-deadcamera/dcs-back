package project

import (
	"errors"
	"testing"
)

// mockStore implements projectStore interface.
type mockStore struct {
	projects map[string]*Project
	scenes   map[string]*Scene
	takes    map[string]*Take
}

func newMockStore() *mockStore {
	return &mockStore{
		projects: make(map[string]*Project),
		scenes:   make(map[string]*Scene),
		takes:    make(map[string]*Take),
	}
}

func (m *mockStore) Create(p *Project) error {
	if _, exists := m.projects[p.ID]; exists {
		return errors.New("duplicate project")
	}
	m.projects[p.ID] = p
	return nil
}

func (m *mockStore) GetByID(id string) (*Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockStore) List() ([]Project, error) {
	var list []Project
	for _, p := range m.projects {
		list = append(list, *p)
	}
	return list, nil
}

func (m *mockStore) Update(id string, updates map[string]interface{}) error {
	p, ok := m.projects[id]
	if !ok {
		return errors.New("project not found")
	}
	if v, ok := updates["name"]; ok {
		p.Name = v.(string)
	}
	if v, ok := updates["description"]; ok {
		p.Description = v.(string)
	}
	if v, ok := updates["metadata"]; ok {
		p.Metadata = v.(string)
	}
	if v, ok := updates["active"]; ok {
		p.Active = v.(bool)
	}
	return nil
}

func (m *mockStore) SoftDelete(id string) error {
	if _, ok := m.projects[id]; !ok {
		return errors.New("project not found")
	}
	delete(m.projects, id)
	return nil
}

func (m *mockStore) CreateScene(sc *Scene) error {
	if _, exists := m.scenes[sc.ID]; exists {
		return errors.New("duplicate scene")
	}
	m.scenes[sc.ID] = sc
	return nil
}

func (m *mockStore) GetSceneByID(id string) (*Scene, error) {
	sc, ok := m.scenes[id]
	if !ok {
		return nil, nil
	}
	return sc, nil
}

func (m *mockStore) ListScenes(projectID string) ([]Scene, error) {
	var list []Scene
	for _, sc := range m.scenes {
		if sc.ProjectID == projectID {
			list = append(list, *sc)
		}
	}
	return list, nil
}

func (m *mockStore) UpdateScene(id string, updates map[string]interface{}) error {
	sc, ok := m.scenes[id]
	if !ok {
		return errors.New("scene not found")
	}
	if v, ok := updates["number"]; ok {
		sc.Number = v.(int)
	}
	if v, ok := updates["name"]; ok {
		sc.Name = v.(string)
	}
	if v, ok := updates["description"]; ok {
		sc.Description = v.(string)
	}
	if v, ok := updates["active"]; ok {
		sc.Active = v.(bool)
	}
	return nil
}

func (m *mockStore) SoftDeleteScene(id string) error {
	if _, ok := m.scenes[id]; !ok {
		return errors.New("scene not found")
	}
	delete(m.scenes, id)
	return nil
}

func (m *mockStore) CreateTake(t *Take) error {
	if _, exists := m.takes[t.ID]; exists {
		return errors.New("duplicate take")
	}
	m.takes[t.ID] = t
	return nil
}

func (m *mockStore) GetTakeByID(id string) (*Take, error) {
	t, ok := m.takes[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *mockStore) ListTakes(sceneID string) ([]Take, error) {
	var list []Take
	for _, t := range m.takes {
		if t.SceneID == sceneID {
			list = append(list, *t)
		}
	}
	return list, nil
}

func (m *mockStore) UpdateTake(id string, updates map[string]interface{}) error {
	t, ok := m.takes[id]
	if !ok {
		return errors.New("take not found")
	}
	if v, ok := updates["video_url"]; ok {
		t.VideoURL = v.(string)
	}
	if v, ok := updates["video_local_url"]; ok {
		t.VideoLocalURL = v.(string)
	}
	if v, ok := updates["status"]; ok {
		t.Status = v.(string)
	}
	if v, ok := updates["active"]; ok {
		t.Active = v.(bool)
	}
	return nil
}

func (m *mockStore) ListActiveTakes(sceneID string) ([]Take, error) {
	var list []Take
	for _, t := range m.takes {
		if t.SceneID == sceneID && t.Active {
			list = append(list, *t)
		}
	}
	return list, nil
}

func (m *mockStore) GetActiveTakeByNumber(sceneID string, number int) (*Take, error) {
	for _, t := range m.takes {
		if t.SceneID == sceneID && t.Number == number && t.Active {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockStore) DeactivateTakesByNumber(sceneID string, number int) error {
	for _, t := range m.takes {
		if t.SceneID == sceneID && t.Number == number && t.Active {
			t.Active = false
		}
	}
	return nil
}

func (m *mockStore) SoftDeleteTake(id string) error {
	if _, ok := m.takes[id]; !ok {
		return errors.New("take not found")
	}
	delete(m.takes, id)
	return nil
}

// ─── Tests ──────────────────────────────────────────────────────

func TestCreateProject(t *testing.T) {
	s := &Service{store: newMockStore()}

	p, err := s.Create(&CreateProjectRequest{
		Name:        "Test Project",
		Description: "A description",
		Metadata:    `{"key":"val"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
	if p.Name != "Test Project" {
		t.Errorf("Name = %q, want %q", p.Name, "Test Project")
	}
	if p.Description != "A description" {
		t.Errorf("Description = %q, want %q", p.Description, "A description")
	}
	if p.Metadata != `{"key":"val"}` {
		t.Errorf("Metadata = %q", p.Metadata)
	}
	if !p.Active {
		t.Error("Active should default to true")
	}
}

func TestCreateProject_EmptyName(t *testing.T) {
	s := &Service{store: newMockStore()}
	p, err := s.Create(&CreateProjectRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID even with empty name")
	}
}

func TestGetByID_Found(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	created, err := svc.Create(&CreateProjectRequest{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("got ID %q, want %q", got.ID, created.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.GetByID("nonexistent")
	if err == nil || err.Error() != "project not found" {
		t.Errorf("expected 'project not found', got %v", err)
	}
}

func TestList_Empty(t *testing.T) {
	s := &Service{store: newMockStore()}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if list == nil {
		t.Error("expected empty slice, not nil")
	}
	if len(list) != 0 {
		t.Errorf("expected 0 projects, got %d", len(list))
	}
}

func TestList_WithProjects(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	svc.Create(&CreateProjectRequest{Name: "A"})
	svc.Create(&CreateProjectRequest{Name: "B"})

	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 projects, got %d", len(list))
	}
}

func TestUpdateProject_AllFields(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	created, _ := svc.Create(&CreateProjectRequest{Name: "original"})

	newName := "updated"
	newDesc := "new desc"
	newMeta := "{}"
	active := false

	updated, err := svc.Update(created.ID, &UpdateProjectRequest{
		Name:        &newName,
		Description: &newDesc,
		Metadata:    &newMeta,
		Active:      &active,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "updated" {
		t.Errorf("Name = %q", updated.Name)
	}
	if updated.Description != "new desc" {
		t.Errorf("Description = %q", updated.Description)
	}
	if updated.Metadata != "{}" {
		t.Errorf("Metadata = %q", updated.Metadata)
	}
	if updated.Active {
		t.Error("Active should be false")
	}
}

func TestUpdateProject_Partial(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	created, _ := svc.Create(&CreateProjectRequest{Name: "original", Description: "desc"})

	newName := "only name changed"
	updated, err := svc.Update(created.ID, &UpdateProjectRequest{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "only name changed" {
		t.Errorf("Name = %q", updated.Name)
	}
	// Description should remain unchanged
	if updated.Description != "desc" {
		t.Errorf("Description = %q, should remain 'desc'", updated.Description)
	}
}

func TestSoftDelete_NotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	err := s.SoftDelete("nonexistent")
	if err == nil || err.Error() != "project not found" {
		t.Errorf("expected 'project not found', got %v", err)
	}
}

func TestCreateScene_ProjectNotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.CreateScene("no-such-project", &CreateSceneRequest{Number: 1})
	if err == nil || err.Error() != "project not found" {
		t.Errorf("expected 'project not found', got %v", err)
	}
}

func TestCreateScene_Success(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})

	sc, err := svc.CreateScene(proj.ID, &CreateSceneRequest{
		Number:      1,
		Name:        "Opening",
		Description: "First scene",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.Number != 1 {
		t.Errorf("Number = %d, want 1", sc.Number)
	}
	if sc.Name != "Opening" {
		t.Errorf("Name = %q", sc.Name)
	}
	if !sc.Active {
		t.Error("Scene should be active by default")
	}
	if sc.ProjectID != proj.ID {
		t.Errorf("ProjectID = %q, want %q", sc.ProjectID, proj.ID)
	}
}

func TestCreateTake_DefaultStatus(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	sc, _ := svc.CreateScene(proj.ID, &CreateSceneRequest{Number: 1})

	take, err := svc.CreateTake(sc.ID, &CreateTakeRequest{Number: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if take.Status != "pending" {
		t.Errorf("Status = %q, want 'pending'", take.Status)
	}
	if !take.Active {
		t.Error("Take should be active by default")
	}
	if take.Number != 1 {
		t.Errorf("Number = %d, want 1", take.Number)
	}
}

func TestCreateTake_ExplicitStatus(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	sc, _ := svc.CreateScene(proj.ID, &CreateSceneRequest{Number: 1})

	take, err := svc.CreateTake(sc.ID, &CreateTakeRequest{Number: 1, Status: "completed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if take.Status != "completed" {
		t.Errorf("Status = %q, want 'completed'", take.Status)
	}
}

func TestCreateTake_SceneNotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.CreateTake("no-such-scene", &CreateTakeRequest{Number: 1})
	if err == nil || err.Error() != "scene not found" {
		t.Errorf("expected 'scene not found', got %v", err)
	}
}

func TestUpdateTake_VideoURL(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	sc, _ := svc.CreateScene(proj.ID, &CreateSceneRequest{Number: 1})
	take, _ := svc.CreateTake(sc.ID, &CreateTakeRequest{Number: 1})

	videoURL := "https://cdn.example.com/video.mp4"
	status := "completed"
	updated, err := svc.UpdateTake(take.ID, &UpdateTakeRequest{
		VideoURL: &videoURL,
		Status:   &status,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.VideoURL != "https://cdn.example.com/video.mp4" {
		t.Errorf("VideoURL = %q", updated.VideoURL)
	}
	if updated.Status != "completed" {
		t.Errorf("Status = %q", updated.Status)
	}
}

func TestGetProjectWithScenes(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	svc.CreateScene(proj.ID, &CreateSceneRequest{Number: 1, Name: "Scene 1"})
	svc.CreateScene(proj.ID, &CreateSceneRequest{Number: 2, Name: "Scene 2"})

	result, err := svc.GetProjectWithScenes(proj.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Project.ID != proj.ID {
		t.Errorf("Project.ID = %q", result.Project.ID)
	}
	if len(result.Scenes) != 2 {
		t.Errorf("expected 2 scenes, got %d", len(result.Scenes))
	}
}

func TestGetSceneWithTakes(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	sc, _ := svc.CreateScene(proj.ID, &CreateSceneRequest{Number: 1})
	svc.CreateTake(sc.ID, &CreateTakeRequest{Number: 1})
	svc.CreateTake(sc.ID, &CreateTakeRequest{Number: 2})

	result, err := svc.GetSceneWithTakes(sc.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Scene.ID != sc.ID {
		t.Errorf("Scene.ID = %q", result.Scene.ID)
	}
	if len(result.Takes) != 2 {
		t.Errorf("expected 2 takes, got %d", len(result.Takes))
	}
}

func TestGetProjectWithScenes_ProjectNotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.GetProjectWithScenes("nonexistent")
	if err == nil || err.Error() != "project not found" {
		t.Errorf("expected 'project not found', got %v", err)
	}
}

func TestGetSceneWithTakes_SceneNotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.GetSceneWithTakes("nonexistent")
	if err == nil || err.Error() != "scene not found" {
		t.Errorf("expected 'scene not found', got %v", err)
	}
}

func TestSoftDeleteScene(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	sc, _ := svc.CreateScene(proj.ID, &CreateSceneRequest{Number: 1})

	if err := svc.SoftDeleteScene(sc.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify deleted
	_, err := svc.GetSceneByID(sc.ID)
	if err == nil || err.Error() != "scene not found" {
		t.Errorf("expected 'scene not found' after delete, got %v", err)
	}
}

func TestSoftDeleteTake(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	sc, _ := svc.CreateScene(proj.ID, &CreateSceneRequest{Number: 1})
	take, _ := svc.CreateTake(sc.ID, &CreateTakeRequest{Number: 1})

	if err := svc.SoftDeleteTake(take.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.GetTakeByID(take.ID)
	if err == nil || err.Error() != "take not found" {
		t.Errorf("expected 'take not found' after delete, got %v", err)
	}
}
