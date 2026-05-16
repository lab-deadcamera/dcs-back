package studio

import (
	"time"

	"dcs-back-v0/internal/studio/generators"
)

// ─── Legacy types (Selection-based) ─────────────────────────────

type Selection struct {
	UserPrompt   string            `json:"userPrompt"`
	ModelID      string            `json:"model_id" binding:"required"`
	Duration     float64           `json:"duration"`
	SoundOn      *bool             `json:"soundOn"`
	AspectRatio  *SelectionField   `json:"aspectRatio"`
	Resolution   *SelectionField   `json:"resolution"`
	CameraMotion *SelectionPrompt  `json:"cameraMotion"`
	Camera       *SelectionPrompt  `json:"camera"`
	Lens         *SelectionPrompt  `json:"lens"`
	ColorGrading *SelectionPrompt  `json:"colorGrading"`
	Genre        *SelectionPrompt  `json:"genre"`
	FirstFrame   *DataRef          `json:"firstFrame"`
	LastFrame    *DataRef          `json:"lastFrame"`
	RefImages    []DataRef         `json:"refImages"`
	RefVideos    []DataRef         `json:"refVideos"`
	RefAudios    []DataRef         `json:"refAudios"`
}

type SelectionField struct {
	Value string `json:"value"`
}

type SelectionPrompt struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
}

type DataRef struct {
	DataUrl string `json:"dataUrl"`
}

// ─── Unified payload types ──────────────────────────────────────

// ContentItem represents a single entry in the content array.
type ContentItem struct {
	Type string `json:"type" binding:"required"` // "text", "image", "video", "audio"
	Text string `json:"text,omitempty"`           // prompt text or asset description
	Name string `json:"name,omitempty"`           // original filename (file types)
	ID   string `json:"id,omitempty"`             // file UUID from the file store
}

// StudioGenerateRequest is the new unified payload for /studio/generate.
type StudioGenerateRequest struct {
	Model         string        `json:"model" binding:"required"`
	Content       []ContentItem `json:"content" binding:"required"`
	Ratio         string        `json:"ratio"`
	Duration      float64       `json:"duration"`
	CameraFixed   *bool         `json:"camerafixed"`
	Seed          string        `json:"seed"`
	Quality       string        `json:"quality"`
	Quantity      int           `json:"quantity"`
	Watermark     *bool         `json:"watermark"`
	Resolution    string        `json:"resolution"`
	GenerateAudio *bool         `json:"generate_audio"`
	ImageMode     string        `json:"image_mode"`
}

// OutputResource represents a single generated output (video, image, audio).
type OutputResource struct {
	URL      string `json:"url"`
	LocalURL string `json:"localUrl,omitempty"`
	Type     string `json:"type"` // "video", "image", "audio"
}

// StudioGenerateResponse is returned by POST /studio/generate.
type StudioGenerateResponse struct {
	TaskID  string           `json:"taskId"`
	Model   string           `json:"model"`
	Status  string           `json:"status"`
	Outputs []OutputResource `json:"outputs,omitempty"`
}

// StudioStatusResponse is returned by GET /studio/status/:taskId.
type StudioStatusResponse struct {
	Status  string           `json:"status"`
	Outputs []OutputResource `json:"outputs,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// ─── Legacy response types ──────────────────────────────────────

type GenerateResponse struct {
	TaskID  string `json:"taskId"`
	ModelID string `json:"modelId"`
	Prompt  string `json:"prompt"`
	Model   string `json:"model"`
	Status  string `json:"status"`
}

type StatusResult struct {
	Status   string      `json:"status"`
	VideoURL string      `json:"videoUrl,omitempty"`
	LocalURL string      `json:"localUrl,omitempty"`
	ImageURL string      `json:"imageUrl,omitempty"`
	Raw      interface{} `json:"raw,omitempty"`
	Error    string      `json:"error,omitempty"`
}

type TaskCancelResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ─── Model Handler Interface (legacy) ───────────────────────────

type ModelHandler interface {
	Matches(modelName string) bool
	Generate(sel *Selection, apiKey, baseURL, endpoint string) (*GenerateResponse, error)
	GetStatus(taskID string, apiKey, baseURL, endpoint string) (*StatusResult, error)
	CancelTask(taskID string, apiKey, baseURL, endpoint string) error
}

// ─── Generator adapter ──────────────────────────────────────────

// GeneratorAdapter wraps a generators.Generator to be used as a ModelHandler.
type GeneratorAdapter struct {
	gen generators.Generator
}

func NewGeneratorAdapter(gen generators.Generator) *GeneratorAdapter {
	return &GeneratorAdapter{gen: gen}
}

func (a *GeneratorAdapter) Matches(modelName string) bool {
	return a.gen.Match(modelName)
}

func (a *GeneratorAdapter) Generate(sel *Selection, apiKey, baseURL, endpoint string) (*GenerateResponse, error) {
	return nil, nil
}

func (a *GeneratorAdapter) GetStatus(taskID, apiKey, baseURL, endpoint string) (*StatusResult, error) {
	result, err := a.gen.GetStatus(taskID, apiKey, baseURL, endpoint)
	if err != nil {
		return nil, err
	}
	sr := &StatusResult{
		Status: result.Status,
		Error:  result.Error,
		Raw:    result.Raw,
	}
	if len(result.Outputs) > 0 {
		sr.VideoURL = result.Outputs[0].URL
		sr.LocalURL = result.Outputs[0].LocalURL
	}
	return sr, nil
}

func (a *GeneratorAdapter) CancelTask(taskID, apiKey, baseURL, endpoint string) error {
	return a.gen.CancelTask(taskID, apiKey, baseURL, endpoint)
}

// ─── Enriched file listing with sync info ───────────────────────

// ModelBrief is a lightweight model reference for sync status responses.
type ModelBrief struct {
	ModelID string `json:"model_id"`
	Name    string `json:"name"`
}

// FileWithSync wraps a file.File with its synced model list.
type FileWithSync struct {
	ID           string       `json:"id"`
	Filename     string       `json:"filename"`
	Path         string       `json:"path"`
	Size         int64        `json:"size"`
	MimeType     string       `json:"mime_type"`
	Category     string       `json:"category"`
	Format       string       `json:"format"`
	Storage      string       `json:"storage"`
	Trashed      bool         `json:"trashed"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	DeletedAt    *time.Time   `json:"deleted_at"`
	SyncedModels []ModelBrief `json:"synced_models"`
}

// CharacterFileWithSync wraps character file data with its synced model list.
type CharacterFileWithSync struct {
	FileID       string       `json:"file_id"`
	Role         string       `json:"role"`
	Filename     string       `json:"filename"`
	URL          string       `json:"url"`
	ThumbnailURL string       `json:"thumbnail_url"`
	MimeType     string       `json:"mime_type"`
	Category     string       `json:"category"`
	Format       string       `json:"format"`
	Size         int64        `json:"size"`
	CreatedAt    time.Time    `json:"created_at"`
	SyncedModels []ModelBrief `json:"synced_models"`
}

// SyncCharacterRequest is the payload for syncing all character assets to a model.
type SyncCharacterRequest struct {
	CharacterID string `json:"character_id" binding:"required"`
	ModelID     string `json:"model_id" binding:"required"`
}

// SyncResultSummary is the result of syncing multiple files to a model.
type SyncResultSummary struct {
	ModelID    string              `json:"model_id"`
	Total      int                 `json:"total"`
	Successful int                 `json:"successful"`
	Failed     int                 `json:"failed"`
	Results    []SyncAssetResponse `json:"results"`
}

// ─── Asset sync ─────────────────────────────────────────────────

type SyncAssetRequest struct {
	ModelID string `json:"model_id" binding:"required"`
	FileID  string `json:"file_id" binding:"required"`
}

type SyncAssetResponse struct {
	ID           string `json:"id"`
	ModelID      string `json:"model_id"`
	FileID       string `json:"file_id"`
	AssetID      string `json:"asset_id,omitempty"`
	AssetGroupID string `json:"asset_group_id,omitempty"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ─── In-memory task tracking ────────────────────────────────────

type TaskRecord struct {
	TaskID    string
	ModelID   string
	ModelName string
	CreatedAt time.Time
	Status    string
	Result    *StatusResult
}

// ─── Generation log types ───────────────────────────────────────

// GenerationLog stores the complete log for a generation task,
// linking the client payload with the AI response via task ID.
type GenerationLog struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"task_id"`
	ModelName     string     `json:"model_name"`
	Request       string     `json:"request"`                     // original client payload (JSON)
	AIResponse    string     `json:"ai_response"`                 // raw AI API response (JSON)
	AICallPayload string     `json:"ai_call_payload,omitempty"`  // payload sent to AI API (JSON)
	Outputs       string     `json:"outputs,omitempty"`
	Status        string     `json:"status"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// ListGenerationLogsRequest holds pagination params for listing logs.
type ListGenerationLogsRequest struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

// ListGenerationLogsResponse holds the paginated response for listing logs.
type ListGenerationLogsResponse struct {
	Logs       []GenerationLog `json:"logs"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}

// PreviewPayloadResponse returns the AI API payload without sending it.
type PreviewPayloadResponse struct {
	Model       string                 `json:"model"`
	Endpoint    string                 `json:"endpoint"`
	Payload     map[string]interface{} `json:"payload"`
	ContentType string                 `json:"content_type"`
}
