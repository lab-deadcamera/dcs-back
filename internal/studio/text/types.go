package text

// ─── Text generation request/response types ─────────────────────

// ContentItem represents a single entry in the content array for text generation.
type ContentItem struct {
	Type string `json:"type" binding:"required"` // "text"
	Text string `json:"text,omitempty"`           // prompt text
	Name string `json:"name,omitempty"`           // not used for text
	ID   string `json:"id,omitempty"`             // not used for text
}

// GenerateRequest is the payload for POST /studio/text/generate.
type GenerateRequest struct {
	Model    string        `json:"model" binding:"required"`
	Content  []ContentItem `json:"content" binding:"required"`
	Seed     string        `json:"seed"`
	Quality  string        `json:"quality"`
	Quantity int           `json:"quantity"`
	// Session tracking
	ProjectID  string `json:"project_id" binding:"required"`
	SceneID    string `json:"scene_id" binding:"required"`
	SceneCode  string `json:"scene_code" binding:"required"`
	TakeNumber int    `json:"take_number" binding:"required"`
	UserID     int    `json:"user_id"`
}

// OutputResource represents a single generated text output.
type OutputResource struct {
	Text string `json:"text"`
}

// GenerateResponse is returned by POST /studio/text/generate.
type GenerateResponse struct {
	TaskID  string           `json:"taskId"`
	Model   string           `json:"model"`
	Status  string           `json:"status"`
	Outputs []OutputResource `json:"outputs,omitempty"`
}

// StatusResponse is returned by GET /studio/text/status/:taskId.
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
