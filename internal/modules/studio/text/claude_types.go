package text

// ─── Episode-level asset assignment ─────────────────────────────

type EpisodeAsset struct {
	Slot    string `json:"slot"`
	AssetID string `json:"assetId"`
	Type    string `json:"type"`
}

type Episode struct {
	Title           string         `json:"title,omitempty"`
	TotalDuration   int            `json:"totalDuration,omitempty"`
	TotalShots      int            `json:"totalShots,omitempty"`
	AssetAssignments []EpisodeAsset `json:"assetAssignments,omitempty"`
}

// ─── Scene-level types ──────────────────────────────────────────

type SceneContinuity struct {
	Location            string   `json:"location"`
	LocationChange      bool     `json:"locationChange"`
	TimeContinuity      string   `json:"timeContinuity"`
	CharactersPresent   []string `json:"charactersPresent"`
	EmotionalCarryover  string   `json:"emotionalCarryover,omitempty"`
	PhysicalCarryover   string   `json:"physicalCarryover,omitempty"`
	WardrobeCarryover   string   `json:"wardrobeCarryover,omitempty"`
	Notes               []string `json:"notes,omitempty"`
}

type Camera struct {
	Lens        string `json:"lens"`
	Framing     string `json:"framing"`
	Movement    string `json:"movement"`
	FPS         int    `json:"fps"`
	Shutter     string `json:"shutter"`
	AspectRatio string `json:"aspectRatio"`
}

type Composition struct {
	FrameMap        string `json:"frameMap"`
	SubjectLock     string `json:"subjectLock"`
	CrossFrameRules string `json:"crossFrameRules"`
	Focus           string `json:"focus"`
	Depth           string `json:"depth"`
}

type BlockingPosition struct {
	SubjectID   string `json:"subjectId"`
	Description string `json:"description"`
}

type Blocking struct {
	Location    string             `json:"location"`
	Movement    string             `json:"movement"`
	Interaction string             `json:"interaction"`
	Positions   []BlockingPosition `json:"positions"`
}

type Acting struct {
	Emotion          string   `json:"emotion"`
	BodyLanguage     string   `json:"bodyLanguage"`
	Dialogue         string   `json:"dialogue"`
	MicroExpressions []string `json:"microExpressions"`
}

type TimelineSegment struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Label string `json:"label"`
}

type TimelineBeat struct {
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Description string `json:"description"`
}

type Timeline struct {
	Duration int               `json:"duration"`
	Segments []TimelineSegment `json:"segments"`
	Beats    []TimelineBeat    `json:"beats"`
}

type Audio struct {
	Dialogue string   `json:"dialogue"`
	Ambient  string   `json:"ambient"`
	Sfx      []string `json:"sfx"`
	Music    bool     `json:"music"`
}

type PromptPair struct {
	En string `json:"en"`
	Zh string `json:"zh,omitempty"`
}

type Render struct {
	Mode   string `json:"mode"`
	Engine string `json:"engine"`
}

type ShotNotes struct {
	Todos    []string `json:"todos"`
	Warnings []string `json:"warnings"`
	Approved bool     `json:"approved"`
}

type ShotReference struct {
	Slot    string `json:"slot"`
	AssetID string `json:"assetId"`
	Type    string `json:"type"`
}

type Shot struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Duration    int         `json:"duration"`
	Start       int         `json:"start"`
	End         int         `json:"end"`
	Camera      Camera      `json:"camera"`
	Composition Composition `json:"composition"`
	Blocking    Blocking    `json:"blocking"`
	Acting      Acting      `json:"acting"`
	Timeline    Timeline    `json:"timeline"`
	Audio       Audio       `json:"audio"`
	References  []ShotReference `json:"references"`
	Prompt      PromptPair  `json:"prompt"`
	Render      Render      `json:"render"`
	Notes       ShotNotes   `json:"notes"`
}

type SceneData struct {
	ScriptNumber   int              `json:"scriptNumber"`
	ScriptLocation string           `json:"scriptLocation"`
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Duration       int              `json:"duration"`
	Start          int              `json:"start"`
	End            int              `json:"end"`
	SceneType      string           `json:"sceneType"`
	Mode           string           `json:"mode"`
	Continuity     SceneContinuity  `json:"continuity"`
	References     []ShotReference  `json:"references"`
	Shots          []Shot           `json:"shots"`
}

type DirectorNotes struct {
	Goal       string   `json:"goal"`
	StyleGuide string   `json:"styleGuide"`
	Warnings   []string `json:"warnings"`
}

// ─── Shots API response (legacy, used by create-shot endpoints) ─

type ShotRecord struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Name   string `json:"name"`
}

// ─── Scene Context (used by both Shot Builder and Proncer) ───────

type SceneContextCharacter struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Slot        string `json:"slot,omitempty"`
}

type SceneContextPreset struct {
	Code   string `json:"code"`
	Label  string `json:"label"`
	Prompt string `json:"prompt,omitempty"`
}

type SceneContextAsset struct {
	ID       string `json:"id,omitempty"`
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
	SceneID      string        `json:"scene_id" binding:"required"`
	ProjectID    string        `json:"project_id" binding:"required"`
	ProjectName  string        `json:"project_name"`
	Model        string        `json:"model"`
	APIModel     string        `json:"api_model"`
	Prompt       string        `json:"prompt" binding:"required"`
	SystemPrompt string        `json:"system_prompt"`
	SkillID      string        `json:"skill_id"`
	UserID       int           `json:"user_id"`
	UserName     string        `json:"user_name"`
	SceneContext *SceneContext `json:"scene_context,omitempty"`
	GenerateZh   bool          `json:"generate_zh"`
}

type ClaudeGenerateShotsResponse struct {
	TaskID string      `json:"taskId"`
	Model  string      `json:"model"`
	Status string      `json:"status"`
	Text   string      `json:"text,omitempty"`
	Episode *Episode   `json:"episode,omitempty"`
	Scenes  []SceneData `json:"scenes,omitempty"`
	DirectorNotes *DirectorNotes `json:"directorNotes,omitempty"`
	AspectRatio string `json:"aspectRatio,omitempty"`
	Mode string        `json:"mode,omitempty"`
}

// ─── Shot Builder Refine ────────────────────────────────────────

// ClaudeRefineShotsRequest is the payload for refining an existing shot
// breakdown. previous_response is the raw JSON returned by generate-shots
// (data.text) and change_request is the user's natural-language instruction.
type ClaudeRefineShotsRequest struct {
	SceneID          string        `json:"scene_id" binding:"required"`
	ProjectID        string        `json:"project_id" binding:"required"`
	ProjectName      string        `json:"project_name"`
	Model            string        `json:"model"`
	APIModel         string        `json:"api_model"`
	PreviousResponse string        `json:"previous_response" binding:"required"`
	ChangeRequest    string        `json:"change_request" binding:"required"`
	SystemPrompt     string        `json:"system_prompt"`
	SkillID          string        `json:"skill_id"`
	UserID           int           `json:"user_id"`
	UserName         string        `json:"user_name"`
	GenerateZh       bool          `json:"generate_zh"`
	SceneContext     *SceneContext `json:"scene_context,omitempty"`
}

type ClaudeRefineShotsResponse struct {
	TaskID string `json:"taskId"`
	Model  string `json:"model"`
	Status string `json:"status"`
	Text   string `json:"text,omitempty"`
}

// shotBuilderMeta carries the request fields needed for failure logging,
// shared by generate-shots and refine-shots.
type shotBuilderMeta struct {
	Mode      string // "generate" | "refine"
	ProjectID string
	SceneID   string
	SkillID   string
	UserID    int
	UserName  string
}

// ListShotBuilderLogsRequest holds pagination and filter params for listing
// failed generate-shots calls.
type ListShotBuilderLogsRequest struct {
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
	ProjectID string `form:"project_id"`
	SceneID   string `form:"scene_id"`
	Mode      string `form:"mode"`
	UserID    int    `form:"user_id"`
	DateFrom  string `form:"date_from"`
	DateTo    string `form:"date_to"`
}

// ─── Proncer ─────────────────────────────────────────────────────

type ClaudeOptimizePromptRequest struct {
	SceneID          string        `json:"scene_id" binding:"required"`
	ProjectID        string        `json:"project_id" binding:"required"`
	Model            string        `json:"model"`
	APIModel         string        `json:"api_model"`
	CurrentPrompt    string        `json:"current_prompt" binding:"required"`
	UserInstructions string        `json:"user_instructions"`
	SystemPrompt     string        `json:"system_prompt"`
	SkillID          string        `json:"skill_id"`
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
