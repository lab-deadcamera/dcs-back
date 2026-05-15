package generators

import (
	"fmt"
	"strings"
)

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
	// Validate checks that the request payload is valid for this generator.
	// Returns an error listing all invalid fields, or nil if valid.
	Validate(req *GeneratorRequest) error
	// Generate submits a generation task and returns the result.
	Generate(req *GeneratorRequest) (*GeneratorResult, error)
	// GetStatus polls the current status of a generation task.
	GetStatus(taskID, apiKey, baseURL, endpoint string) (*GeneratorResult, error)
	// CancelTask cancels a running generation task.
	CancelTask(taskID, apiKey, baseURL, endpoint string) error
}

// ─── Common validation ──────────────────────────────────────────

// ValidationError accumulates multiple validation errors.
type ValidationError struct {
	Fields []string
}

func (e *ValidationError) Error() string {
	return "validation failed: " + strings.Join(e.Fields, "; ")
}

func (e *ValidationError) Add(field, msg string) {
	e.Fields = append(e.Fields, field+": "+msg)
}

func (e *ValidationError) HasErrors() bool {
	return len(e.Fields) > 0
}

// validateCommon checks fields shared across all generators.
func validateCommon(req *GeneratorRequest) *ValidationError {
	errs := &ValidationError{}

	if strings.TrimSpace(req.Model) == "" {
		errs.Add("model", "is required")
	}
	if len(req.Content) == 0 {
		errs.Add("content", "must have at least one item")
	} else {
		hasText := false
		for i, item := range req.Content {
			if item.Type == "" {
				errs.Add(fmt.Sprintf("content[%d]", i), "type is required (text, image, video, audio)")
			}
			if item.Type == "text" && strings.TrimSpace(item.Text) != "" {
				hasText = true
			}
		}
		if !hasText {
			errs.Add("content", "must include at least one text item with a prompt")
		}
	}

	return errs
}

// validRatios lists supported aspect ratios.
var validRatios = map[string]bool{
	"16:9": true, "9:16": true, "1:1": true, "4:3": true, "3:4": true,
	"21:9": true, "9:21": true, "2.35:1": true,
}

// validResolutionsVideo lists supported video resolutions.
var validResolutionsVideo = map[string]bool{
	"480p": true, "720p": true, "1080p": true,
}

// validResolutionsImage lists supported image resolutions.
var validResolutionsImage = map[string]bool{
	"2K": true, "1080p": true, "720p": true,
}
