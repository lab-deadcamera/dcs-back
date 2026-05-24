package preset

import "time"

type Group struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description,omitempty"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type Preset struct {
	ID        string     `json:"id"`
	GroupID   string     `json:"group_id"`
	Code      string     `json:"code"`
	Label     string     `json:"label"`
	Prompt    string     `json:"prompt"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`
}

type UpdateGroupRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Active      *bool   `json:"active"`
}

type CreatePresetRequest struct {
	GroupID string `json:"group_id" binding:"required"`
	Code    string `json:"code" binding:"required"`
	Label   string `json:"label" binding:"required"`
	Prompt  string `json:"prompt" binding:"required"`
}

type UpdatePresetRequest struct {
	Code   *string `json:"code"`
	Label  *string `json:"label"`
	Prompt *string `json:"prompt"`
	Active *bool   `json:"active"`
}
