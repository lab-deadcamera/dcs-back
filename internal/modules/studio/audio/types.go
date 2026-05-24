package audio

// ─── Audio generation request/response types ─────────────────────

// ContentItem represents a single entry in the content array for audio generation.
type ContentItem struct {
	Type string `json:"type" binding:"required"` // "text", "audio"
	Text string `json:"text,omitempty"`          // prompt text or asset description
	Name string `json:"name,omitempty"`          // original filename (file types)
	ID   string `json:"id,omitempty"`            // file UUID from the file store
}

// GenerateRequest is the payload for POST /studio/audio/generate.
type GenerateRequest struct {
	Model    string        `json:"model" binding:"required"`
	Content  []ContentItem `json:"content" binding:"required"`
	Duration float64       `json:"duration"`
	Seed     string        `json:"seed"`
	Quality  string        `json:"quality"`
	// Session tracking
	ProjectID  string `json:"project_id" binding:"required"`
	SceneID    string `json:"scene_id" binding:"required"`
	SceneCode  string `json:"scene_code" binding:"required"`
	TakeNumber int    `json:"take_number" binding:"required"`
	UserID     int    `json:"user_id"`
}

// OutputResource represents a single generated audio output.
type OutputResource struct {
	URL      string `json:"url"`
	LocalURL string `json:"localUrl,omitempty"`
}

// GenerateResponse is returned by POST /studio/audio/generate.
type GenerateResponse struct {
	TaskID  string           `json:"taskId"`
	Model   string           `json:"model"`
	Status  string           `json:"status"`
	Outputs []OutputResource `json:"outputs,omitempty"`
}

// StatusResponse is returned by GET /studio/audio/status/:taskId.
type StatusResponse struct {
	Status   string           `json:"status"`
	Outputs  []OutputResource `json:"outputs,omitempty"`
	Error    string           `json:"error,omitempty"`
	Raw      interface{}      `json:"raw,omitempty"`
	Progress interface{}      `json:"progress,omitempty"`
}

// PreviewPayloadResponse returns the AI API payload without sending it.
type PreviewPayloadResponse struct {
	Model       string                 `json:"model"`
	Endpoint    string                 `json:"endpoint"`
	Payload     map[string]interface{} `json:"payload"`
	ContentType string                 `json:"content_type"`
}
