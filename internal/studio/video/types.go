package video

// ─── Video generation request/response types ─────────────────────

// ContentItem represents a single entry in the content array for video generation.
type ContentItem struct {
	Type string `json:"type" binding:"required"` // "text", "image", "video", "audio"
	Text string `json:"text,omitempty"`           // prompt text or asset description
	Name string `json:"name,omitempty"`           // original filename (file types)
	ID   string `json:"id,omitempty"`             // file UUID from the file store
}

// GenerateRequest is the payload for POST /studio/video/generate.
type GenerateRequest struct {
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
	// Session tracking
	ProjectID  string `json:"project_id" binding:"required"`
	SceneID    string `json:"scene_id" binding:"required"`
	SceneCode  string `json:"scene_code" binding:"required"`
	TakeNumber int    `json:"take_number" binding:"required"`
	UserID     int    `json:"user_id"`
}

// OutputResource represents a single generated video output.
type OutputResource struct {
	URL      string `json:"url"`
	LocalURL string `json:"localUrl,omitempty"`
}

// GenerateResponse is returned by POST /studio/video/generate.
type GenerateResponse struct {
	TaskID  string           `json:"taskId"`
	Model   string           `json:"model"`
	Status  string           `json:"status"`
	Outputs []OutputResource `json:"outputs,omitempty"`
}

// StatusResponse is returned by GET /studio/video/status/:taskId.
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
