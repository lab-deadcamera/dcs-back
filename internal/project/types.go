package project

import "time"

// ─── Project ────────────────────────────────────────────────────

type Project struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Metadata    string     `json:"metadata,omitempty"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
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

// ─── Scene ──────────────────────────────────────────────────────

type Scene struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Number      int        `json:"number"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Active      bool       `json:"active"`
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

// ─── Take ───────────────────────────────────────────────────────

type Take struct {
	ID            string     `json:"id"`
	SceneID       string     `json:"scene_id"`
	Number        int        `json:"number"`
	VideoURL      string     `json:"video_url"`
	VideoLocalURL string     `json:"video_local_url,omitempty"`
	Status        string     `json:"status"`
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
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
}

// ─── Combined responses ─────────────────────────────────────────

type ProjectWithScenes struct {
	Project Project `json:"project"`
	Scenes  []Scene `json:"scenes"`
}

type SceneWithTakes struct {
	Scene Scene  `json:"scene"`
	Takes []Take `json:"takes"`
}
