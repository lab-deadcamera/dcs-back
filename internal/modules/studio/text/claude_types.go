package text

// ─── Scene Context (used by both Shot Builder and Proncer) ───────

type SceneContextCharacter struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type SceneContextPreset struct {
	Code   string `json:"code"`
	Label  string `json:"label"`
	Prompt string `json:"prompt,omitempty"`
}

type SceneContextAsset struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
}

type SceneContext struct {
	Description string                `json:"description,omitempty"`
	Characters  []SceneContextCharacter `json:"characters,omitempty"`
	Presets     []SceneContextPreset    `json:"presets,omitempty"`
	Assets      []SceneContextAsset     `json:"assets,omitempty"`
}

// ─── Shot Builder ────────────────────────────────────────────────

type ClaudeGenerateShotsRequest struct {
	SceneID    string        `json:"scene_id" binding:"required"`
	ProjectID  string        `json:"project_id" binding:"required"`
	Model      string        `json:"model" binding:"required"`
	Prompt     string        `json:"prompt" binding:"required"`
	SystemPrompt string      `json:"system_prompt"`
	UserID     int           `json:"user_id"`
	UserName   string        `json:"user_name"`
	SceneContext *SceneContext `json:"scene_context,omitempty"`
}

type ClaudeGenerateShotsResponse struct {
	TaskID string `json:"taskId"`
	Model  string `json:"model"`
	Status string `json:"status"`
	Text   string `json:"text,omitempty"`
}

// ─── Proncer ─────────────────────────────────────────────────────

type ClaudeOptimizePromptRequest struct {
	SceneID          string        `json:"scene_id" binding:"required"`
	ProjectID        string        `json:"project_id" binding:"required"`
	Model            string        `json:"model" binding:"required"`
	CurrentPrompt    string        `json:"current_prompt" binding:"required"`
	UserInstructions string        `json:"user_instructions"`
	SystemPrompt     string        `json:"system_prompt"`
	UserID           int           `json:"user_id"`
	UserName         string        `json:"user_name"`
	ShotContext      *ShotContext  `json:"shot_context,omitempty"`
	SceneContext     *SceneContext `json:"scene_context,omitempty"`
}

type ShotContext struct {
	ShotName        string `json:"shot_name,omitempty"`
	ShotDescription string `json:"shot_description,omitempty"`
}

type ClaudeOptimizePromptResponse struct {
	TaskID          string   `json:"taskId"`
	Model           string   `json:"model"`
	Status          string   `json:"status"`
	OptimizedPrompt string   `json:"optimized_prompt"`
	Suggestions     []string `json:"suggestions,omitempty"`
	ChangesMade     []string `json:"changes_made,omitempty"`
	RawText         string   `json:"raw_text,omitempty"`
}
