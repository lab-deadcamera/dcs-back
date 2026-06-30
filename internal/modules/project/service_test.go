package project

import (
	"errors"
	"testing"
)

// mockStore implements projectStore interface.
type mockStore struct {
	projects map[string]*Project
	chapters map[string]*Chapter
	scenes   map[string]*Scene
	shots    map[string]*Shot
	takes    map[string]*Take
}

func newMockStore() *mockStore {
	return &mockStore{
		projects: make(map[string]*Project),
		chapters: make(map[string]*Chapter),
		scenes:   make(map[string]*Scene),
		shots:    make(map[string]*Shot),
		takes:    make(map[string]*Take),
	}
}

// ── Projects ───────────────────────────────────────────────────

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

func (m *mockStore) ListAll() ([]Project, error) {
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

// ── Chapters ───────────────────────────────────────────────────

func (m *mockStore) CreateChapter(c *Chapter) error {
	if _, exists := m.chapters[c.ID]; exists {
		return errors.New("duplicate chapter")
	}
	m.chapters[c.ID] = c
	return nil
}

func (m *mockStore) GetChapterByID(id string) (*Chapter, error) {
	c, ok := m.chapters[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *mockStore) ListChapters(projectID string) ([]Chapter, error) {
	var list []Chapter
	for _, c := range m.chapters {
		if c.ProjectID == projectID {
			list = append(list, *c)
		}
	}
	return list, nil
}

func (m *mockStore) UpdateChapter(id string, updates map[string]interface{}) error {
	c, ok := m.chapters[id]
	if !ok {
		return errors.New("chapter not found")
	}
	if v, ok := updates["number"]; ok {
		c.Number = v.(int)
	}
	if v, ok := updates["name"]; ok {
		c.Name = v.(string)
	}
	if v, ok := updates["description"]; ok {
		c.Description = v.(string)
	}
	if v, ok := updates["active"]; ok {
		c.Active = v.(bool)
	}
	return nil
}

func (m *mockStore) SoftDeleteChapter(id string) error {
	if _, ok := m.chapters[id]; !ok {
		return errors.New("chapter not found")
	}
	delete(m.chapters, id)
	return nil
}

// ── Scenes ─────────────────────────────────────────────────────

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

func (m *mockStore) ListScenes(chapterID string) ([]Scene, error) {
	var list []Scene
	for _, sc := range m.scenes {
		if sc.ChapterID == chapterID {
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

// ── Shots ──────────────────────────────────────────────────────

func (m *mockStore) CreateShot(sh *Shot) error {
	if _, exists := m.shots[sh.ID]; exists {
		return errors.New("duplicate shot")
	}
	m.shots[sh.ID] = sh
	return nil
}

func (m *mockStore) GetShotByID(id string) (*Shot, error) {
	sh, ok := m.shots[id]
	if !ok {
		return nil, nil
	}
	return sh, nil
}

func (m *mockStore) ListShots(sceneID string) ([]Shot, error) {
	var list []Shot
	for _, sh := range m.shots {
		if sh.SceneID == sceneID {
			list = append(list, *sh)
		}
	}
	return list, nil
}

func (m *mockStore) UpdateShot(id string, updates map[string]interface{}) error {
	sh, ok := m.shots[id]
	if !ok {
		return errors.New("shot not found")
	}
	if v, ok := updates["number"]; ok {
		sh.Number = v.(int)
	}
	if v, ok := updates["name"]; ok {
		sh.Name = v.(string)
	}
	if v, ok := updates["description"]; ok {
		sh.Description = v.(string)
	}
	if v, ok := updates["active"]; ok {
		sh.Active = v.(bool)
	}
	return nil
}

func (m *mockStore) SoftDeleteShot(id string) error {
	if _, ok := m.shots[id]; !ok {
		return errors.New("shot not found")
	}
	delete(m.shots, id)
	return nil
}

// ── Takes ──────────────────────────────────────────────────────

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

func (m *mockStore) ListTakes(shotID string) ([]Take, error) {
	var list []Take
	for _, t := range m.takes {
		if t.ShotID == shotID {
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

func (m *mockStore) ListActiveTakes(shotID string) ([]Take, error) {
	var list []Take
	for _, t := range m.takes {
		if t.ShotID == shotID && t.Active {
			list = append(list, *t)
		}
	}
	return list, nil
}

func (m *mockStore) GetActiveTakeByNumber(shotID string, number int) (*Take, error) {
	for _, t := range m.takes {
		if t.ShotID == shotID && t.Number == number && t.Active {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockStore) DeactivateFinalsByNumber(shotID string, number int) error {
	for _, t := range m.takes {
		if t.ShotID == shotID && t.Number == number && t.Final {
			t.Final = false
		}
	}
	return nil
}

func (m *mockStore) DeactivateTakesByNumber(shotID string, number int) error {
	for _, t := range m.takes {
		if t.ShotID == shotID && t.Number == number && t.Active {
			t.Active = false
		}
	}
	return nil
}

func (m *mockStore) GetTakeByVideoURL(shotID string, videoURL string) (*Take, error) {
	for _, t := range m.takes {
		if t.ShotID == shotID && t.VideoURL == videoURL {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockStore) GetPendingTakeByNumber(shotID string, number int) (*Take, error) {
	var pending *Take
	for _, t := range m.takes {
		if t.ShotID == shotID && t.Number == number && t.Status == "pending" {
			if pending == nil || t.CreatedAt.After(pending.CreatedAt) {
				pending = t
			}
		}
	}
	return pending, nil
}

func (m *mockStore) SoftDeleteTake(id string) error {
	if _, ok := m.takes[id]; !ok {
		return errors.New("take not found")
	}
	delete(m.takes, id)
	return nil
}

// ── Scene Assignments ──────────────────────────────────────────

func (m *mockStore) GetScenePresets(sceneID string) ([]ScenePresetAssignment, error) {
	return nil, nil
}

func (m *mockStore) GetSceneCharacters(sceneID string) ([]SceneCharacterAssignment, error) {
	return nil, nil
}

func (m *mockStore) GetSceneAssets(sceneID string) ([]SceneAssetAssignment, error) {
	return nil, nil
}

func (m *mockStore) AssignPresetToScene(sceneID, presetID string) (string, error) {
	return "", nil
}

func (m *mockStore) AssignCharacterToScene(sceneID, characterID string) (string, error) {
	return "", nil
}

func (m *mockStore) AssignAssetToScene(sceneID, fileID string) (string, error) {
	return "", nil
}

func (m *mockStore) RemoveScenePreset(assignmentID string) error {
	return nil
}

func (m *mockStore) RemoveSceneCharacter(assignmentID string) error {
	return nil
}

func (m *mockStore) RemoveSceneAsset(assignmentID string) error {
	return nil
}

// ── Shot Resources (mock) ─────────────────────────────────────

func (m *mockStore) GetShotCharacters(shotID string) ([]ShotCharacterAssignment, error) {
	return nil, nil
}

func (m *mockStore) GetShotAssets(shotID string) ([]ShotAssetAssignment, error) {
	return nil, nil
}

func (m *mockStore) GetShotPresets(shotID string) ([]ShotPresetAssignment, error) {
	return nil, nil
}

func (m *mockStore) AssignCharacterToShot(shotID, characterID string) (string, error) {
	return "", nil
}

func (m *mockStore) AssignAssetToShot(shotID, fileID, slot string) (string, error) {
	return "", nil
}

func (m *mockStore) AssignPresetToShot(shotID, presetID string) (string, error) {
	return "", nil
}

func (m *mockStore) RemoveShotCharacter(assignmentID string) error {
	return nil
}

func (m *mockStore) RemoveShotAsset(assignmentID string) error {
	return nil
}

func (m *mockStore) RemoveShotPreset(assignmentID string) error {
	return nil
}

func (m *mockStore) UpdateShotModel(shotID, modelID string) error {
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

// ─── Chapter Tests ──────────────────────────────────────────────

func TestCreateChapter_Success(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})

	c, err := svc.CreateChapter(proj.ID, &CreateChapterRequest{
		Number:      1,
		Name:        "Chapter 1",
		Description: "First chapter",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Number != 1 {
		t.Errorf("Number = %d, want 1", c.Number)
	}
	if c.Name != "Chapter 1" {
		t.Errorf("Name = %q", c.Name)
	}
	if !c.Active {
		t.Error("Chapter should be active by default")
	}
	if c.ProjectID != proj.ID {
		t.Errorf("ProjectID = %q, want %q", c.ProjectID, proj.ID)
	}
}

func TestCreateChapter_ProjectNotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.CreateChapter("nonexistent", &CreateChapterRequest{Number: 1})
	if err == nil || err.Error() != "project not found" {
		t.Errorf("expected 'project not found', got %v", err)
	}
}

func TestListChapters(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1, Name: "Ch1"})
	svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 2, Name: "Ch2"})

	chapters, err := svc.ListChapters(proj.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(chapters))
	}
}

func TestSoftDeleteChapter(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	c, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})

	if err := svc.SoftDeleteChapter(c.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.GetChapterByID(c.ID)
	if err == nil || err.Error() != "chapter not found" {
		t.Errorf("expected 'chapter not found' after delete, got %v", err)
	}
}

// ─── Scene Tests (adapted) ──────────────────────────────────────

func TestCreateScene_ChapterNotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.CreateScene("no-such-chapter", &CreateSceneRequest{Number: 1})
	if err == nil || err.Error() != "chapter not found" {
		t.Errorf("expected 'chapter not found', got %v", err)
	}
}

func TestCreateScene_Success(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1, Name: "Ch1"})

	sc, err := svc.CreateScene(ch.ID, &CreateSceneRequest{
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
	if sc.ChapterID != ch.ID {
		t.Errorf("ChapterID = %q, want %q", sc.ChapterID, ch.ID)
	}
	if sc.ProjectID != proj.ID {
		t.Errorf("ProjectID = %q, want %q", sc.ProjectID, proj.ID)
	}
}

// ─── Shot Tests ─────────────────────────────────────────────────

func TestCreateShot_Success(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})
	sc, _ := svc.CreateScene(ch.ID, &CreateSceneRequest{Number: 1})

	sh, err := svc.CreateShot(sc.ID, &CreateShotRequest{
		Number:      1,
		Name:        "Close-up",
		Description: "Close-up shot",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sh.Number != 1 {
		t.Errorf("Number = %d, want 1", sh.Number)
	}
	if sh.Name != "Close-up" {
		t.Errorf("Name = %q", sh.Name)
	}
	if !sh.Active {
		t.Error("Shot should be active by default")
	}
	if sh.SceneID != sc.ID {
		t.Errorf("SceneID = %q, want %q", sh.SceneID, sc.ID)
	}
}

func TestCreateShot_SceneNotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.CreateShot("no-such-scene", &CreateShotRequest{Number: 1})
	if err == nil || err.Error() != "scene not found" {
		t.Errorf("expected 'scene not found', got %v", err)
	}
}

func TestListShots(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})
	sc, _ := svc.CreateScene(ch.ID, &CreateSceneRequest{Number: 1})
	svc.CreateShot(sc.ID, &CreateShotRequest{Number: 1, Name: "Shot1"})
	svc.CreateShot(sc.ID, &CreateShotRequest{Number: 2, Name: "Shot2"})

	shots, err := svc.ListShots(sc.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shots) != 2 {
		t.Errorf("expected 2 shots, got %d", len(shots))
	}
}

func TestSoftDeleteShot(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})
	sc, _ := svc.CreateScene(ch.ID, &CreateSceneRequest{Number: 1})
	sh, _ := svc.CreateShot(sc.ID, &CreateShotRequest{Number: 1})

	if err := svc.SoftDeleteShot(sh.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.GetShotByID(sh.ID)
	if err == nil || err.Error() != "shot not found" {
		t.Errorf("expected 'shot not found' after delete, got %v", err)
	}
}

// ─── Take Tests (adapted) ───────────────────────────────────────

func TestCreateTake_DefaultStatus(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})
	sc, _ := svc.CreateScene(ch.ID, &CreateSceneRequest{Number: 1})
	sh, _ := svc.CreateShot(sc.ID, &CreateShotRequest{Number: 1})

	take, err := svc.CreateTake(sh.ID, &CreateTakeRequest{Number: 1})
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
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})
	sc, _ := svc.CreateScene(ch.ID, &CreateSceneRequest{Number: 1})
	sh, _ := svc.CreateShot(sc.ID, &CreateShotRequest{Number: 1})

	take, err := svc.CreateTake(sh.ID, &CreateTakeRequest{Number: 1, Status: "completed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if take.Status != "completed" {
		t.Errorf("Status = %q, want 'completed'", take.Status)
	}
}

func TestCreateTake_ShotNotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.CreateTake("no-such-shot", &CreateTakeRequest{Number: 1})
	if err == nil || err.Error() != "shot not found" {
		t.Errorf("expected 'shot not found', got %v", err)
	}
}

func TestUpdateTake_VideoURL(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})
	sc, _ := svc.CreateScene(ch.ID, &CreateSceneRequest{Number: 1})
	sh, _ := svc.CreateShot(sc.ID, &CreateShotRequest{Number: 1})
	take, _ := svc.CreateTake(sh.ID, &CreateTakeRequest{Number: 1})

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

func TestGetProjectWithChapters(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1, Name: "Ch1"})
	svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 2, Name: "Ch2"})

	result, err := svc.GetProjectWithChapters(proj.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Project.ID != proj.ID {
		t.Errorf("Project.ID = %q", result.Project.ID)
	}
	if len(result.Chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(result.Chapters))
	}
}

func TestGetShotWithTakes(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})
	sc, _ := svc.CreateScene(ch.ID, &CreateSceneRequest{Number: 1})
	sh, _ := svc.CreateShot(sc.ID, &CreateShotRequest{Number: 1})
	svc.CreateTake(sh.ID, &CreateTakeRequest{Number: 1})
	svc.CreateTake(sh.ID, &CreateTakeRequest{Number: 2})

	result, err := svc.GetShotWithTakes(sh.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Shot.ID != sh.ID {
		t.Errorf("Shot.ID = %q", result.Shot.ID)
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

func TestGetShotWithTakes_ShotNotFound(t *testing.T) {
	s := &Service{store: newMockStore()}
	_, err := s.GetShotWithTakes("nonexistent")
	if err == nil || err.Error() != "shot not found" {
		t.Errorf("expected 'shot not found', got %v", err)
	}
}

func TestSoftDeleteScene(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})
	sc, _ := svc.CreateScene(ch.ID, &CreateSceneRequest{Number: 1})

	if err := svc.SoftDeleteScene(sc.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.GetSceneByID(sc.ID)
	if err == nil || err.Error() != "scene not found" {
		t.Errorf("expected 'scene not found' after delete, got %v", err)
	}
}

func TestSoftDeleteTake(t *testing.T) {
	m := newMockStore()
	svc := &Service{store: m}

	proj, _ := svc.Create(&CreateProjectRequest{Name: "test"})
	ch, _ := svc.CreateChapter(proj.ID, &CreateChapterRequest{Number: 1})
	sc, _ := svc.CreateScene(ch.ID, &CreateSceneRequest{Number: 1})
	sh, _ := svc.CreateShot(sc.ID, &CreateShotRequest{Number: 1})
	take, _ := svc.CreateTake(sh.ID, &CreateTakeRequest{Number: 1})

	if err := svc.SoftDeleteTake(take.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.GetTakeByID(take.ID)
	if err == nil || err.Error() != "take not found" {
		t.Errorf("expected 'take not found' after delete, got %v", err)
	}
}
