package generators

// ContentItem is a resolved content item passed to a generator.
// For file-type items, DataURL is already populated by the service layer.
type ContentItem struct {
	Type   string `json:"type"`   // "text", "image", "video", "audio"
	Text   string `json:"text"`   // prompt text or asset description
	Name   string `json:"name"`   // original filename (file types only)
	ID     string `json:"id"`     // file UUID from the file store (file types only)
	DataURL string `json:"-"`    // resolved data URL (populated by service)
}

// GeneratorRequest is the unified request payload for all generators.
type GeneratorRequest struct {
	Model         string
	Content       []ContentItem
	Ratio         string
	Duration      int
	CameraFixed   bool
	Seed          string
	Quality       string
	Quantity      int
	Watermark     bool
	Resolution    string
	GenerateAudio bool
	ImageMode     string
	APIKey        string
	BaseURL       string
	Endpoint      string
}

// OutputResource represents a single generated output resource.
type OutputResource struct {
	URL      string `json:"url"`
	LocalURL string `json:"localUrl,omitempty"`
	Type     string `json:"type"` // "video", "image", "audio"
}

// GeneratorResult is the response returned by a generator.
type GeneratorResult struct {
	TaskID  string           `json:"taskId"`
	Model   string           `json:"model"`
	Status  string           `json:"status"`
	Outputs []OutputResource `json:"outputs,omitempty"`
	Raw     interface{}      `json:"raw,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// Generator defines the interface that all model generators must implement.
type Generator interface {
	// Name returns a human-readable name for this generator.
	Name() string
	// Match returns true if this generator should handle the given model name.
	Match(modelName string) bool
	// Generate submits a generation task and returns the result.
	Generate(req *GeneratorRequest) (*GeneratorResult, error)
	// GetStatus polls the current status of a generation task.
	GetStatus(taskID, apiKey, baseURL, endpoint string) (*GeneratorResult, error)
	// CancelTask cancels a running generation task.
	CancelTask(taskID, apiKey, baseURL, endpoint string) error
}
