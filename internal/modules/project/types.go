package project

import "time"

// ─── Project ────────────────────────────────────────────────────

type Project struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Metadata     string     `json:"metadata"`
	Active       bool       `json:"active"`
	ChapterCount int        `json:"chapter_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Metadata    *string `json:"metadata"`
	Active      *bool   `json:"active"`
}

// ─── Chapter ────────────────────────────────────────────────────

type Chapter struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Number      int        `json:"number"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Active      bool       `json:"active"`
	SceneCount  int        `json:"scene_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type CreateChapterRequest struct {
	Number      int    `json:"number" binding:"required"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateChapterRequest struct {
	Number      *int    `json:"number"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Active      *bool   `json:"active"`
}

// ─── Scene ──────────────────────────────────────────────────────

type Scene struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	ChapterID   string     `json:"chapter_id"`
	Number      int        `json:"number"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Active      bool       `json:"active"`
	ShotCount   int        `json:"shot_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type CreateSceneRequest struct {
	Number      int    `json:"number" binding:"required"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateSceneRequest struct {
	Number      *int    `json:"number"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Active      *bool   `json:"active"`
}

// ─── Shot ───────────────────────────────────────────────────────

type Shot struct {
	ID              string     `json:"id"`
	SceneID         string     `json:"scene_id"`
	Number          int        `json:"number"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Active          bool       `json:"active"`
	TakeCount       int        `json:"take_count"`
	AspectRatio     *string    `json:"aspect_ratio,omitempty"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

type CreateShotRequest struct {
	Number          int     `json:"number" binding:"required"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	AspectRatio     *string `json:"aspect_ratio,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
}

type UpdateShotRequest struct {
	Number          *int    `json:"number"`
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	Active          *bool   `json:"active"`
	AspectRatio     *string `json:"aspect_ratio,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
}

// ─── Take ───────────────────────────────────────────────────────

type Take struct {
	ID             string     `json:"id"`
	SceneID        string     `json:"scene_id"`
	ShotID         string     `json:"shot_id"`
	Number         int        `json:"number"`
	VideoURL       string     `json:"video_url"`
	VideoLocalURL  string     `json:"video_local_url"`
	Status         string     `json:"status"`
	Active         bool       `json:"active"`
	Final          bool       `json:"final"`
	FinalizedAt    *time.Time `json:"finalized_at"`
	TaskID         string     `json:"task_id,omitempty"`
	RequestPayload string     `json:"request_payload,omitempty"`
	Rating         int        `json:"rating"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

type CreateTakeRequest struct {
	Number int    `json:"number" binding:"required"`
	Status string `json:"status"`
}

type UpdateTakeRequest struct {
	VideoURL      *string `json:"video_url"`
	VideoLocalURL *string `json:"video_local_url"`
	Status        *string `json:"status"`
	Active        *bool   `json:"active"`
	Final         *bool   `json:"final"`
	TaskID        *string `json:"task_id"`
	Rating        *int    `json:"rating"`
}

// ─── Combined responses ─────────────────────────────────────────

type ProjectWithChapters struct {
	Project  Project             `json:"project"`
	Chapters []ChapterWithScenes `json:"chapters"`
}

type ChapterWithScenes struct {
	Chapter Chapter          `json:"chapter"`
	Scenes  []SceneWithShots `json:"scenes"`
}

type SceneWithShots struct {
	Scene Scene           `json:"scene"`
	Shots []ShotWithTakes `json:"shots"`
}

type ShotWithTakes struct {
	Shot  Shot   `json:"shot"`
	Takes []Take `json:"takes"`
}

// Legacy aliases for backward compatibility during migration

type SceneWithTakes struct {
	Scene Scene  `json:"scene"`
	Takes []Take `json:"takes"`
}

type ProjectWithScenes struct {
	Project Project `json:"project"`
	Scenes  []Scene `json:"scenes"`
}

// ─── Scene Assignments ─────────────────────────────────────────

type ScenePresetAssignment struct {
	ID        string    `json:"id"`
	SceneID   string    `json:"scene_id"`
	PresetID  string    `json:"preset_id"`
	Code      string    `json:"code"`
	Label     string    `json:"label"`
	GroupSlug string    `json:"group_slug"`
	CreatedAt time.Time `json:"created_at"`
}

type SceneCharacterAssignment struct {
	ID          string    `json:"id"`
	SceneID     string    `json:"scene_id"`
	CharacterID string    `json:"character_id"`
	Name        string    `json:"name"`
	Slot        string    `json:"slot"`
	FileID      string    `json:"file_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type SceneAssetAssignment struct {
	ID        string    `json:"id"`
	SceneID   string    `json:"scene_id"`
	FileID    string    `json:"file_id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
}

type SceneAssignments struct {
	Presets    []ScenePresetAssignment    `json:"presets"`
	Characters []SceneCharacterAssignment `json:"characters"`
	Assets     []SceneAssetAssignment     `json:"assets"`
}



