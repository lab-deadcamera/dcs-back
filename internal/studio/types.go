package studio

import "time"

// ─── Request / Response ─────────────────────────────────────────

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

type GenerateResponse struct {
	TaskID  string `json:"taskId"`
	ModelID string `json:"modelId"`
	Prompt  string `json:"prompt"`
	Model   string `json:"model"`
	Status  string `json:"status"`
}

type StatusResult struct {
	Status   string `json:"status"`
	VideoURL string `json:"videoUrl,omitempty"`
	LocalURL string `json:"localUrl,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
	Raw      interface{} `json:"raw,omitempty"`
	Error    string `json:"error,omitempty"`
}

type TaskCancelResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ─── Model Handler Interface ────────────────────────────────────

type ModelHandler interface {
	// Matches returns true if this handler should be used for the given model
	Matches(modelName string) bool
	// Generate submits a generation task and returns the task ID
	Generate(sel *Selection, apiKey, baseURL, endpoint string) (*GenerateResponse, error)
	// GetStatus polls the task status
	GetStatus(taskID string, apiKey, baseURL, endpoint string) (*StatusResult, error)
	// CancelTask cancels a running task
	CancelTask(taskID string, apiKey, baseURL, endpoint string) error
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
