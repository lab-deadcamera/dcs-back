// Package image defines the domain contracts for image generation.
//
// ImageGenerator is the interface that all image generators must implement.
// ImageRequest/ImageResult define the data it receives and produces at the domain level.
// Internally, generators work with shared types from the generators package.
package image

import "dcs-back-v0/internal/studio"

// ImageGenerator defines the contract that all image generators must satisfy.
type ImageGenerator interface {
	// Name returns a human-readable name for this generator.
	Name() string
	// Match returns true if this generator should handle the given model name.
	Match(modelName string) bool
	// Validate checks that the request payload is valid for this generator.
	Validate(req *studio.GeneratorRequest) error
	// Generate submits a generation task and returns the result.
	Generate(req *studio.GeneratorRequest) (*studio.GeneratorResult, error)
	// GetStatus polls the current status of a generation task.
	GetStatus(taskID, apiKey, baseURL, endpoint string) (*studio.GeneratorResult, error)
	// CancelTask cancels a running generation task.
	CancelTask(taskID, apiKey, baseURL, endpoint string) error
	// BuildPayload returns the raw API payload for preview purposes.
	BuildPayload(req *studio.GeneratorRequest) map[string]interface{}
	// ContentType returns "image".
	ContentType() string
}

// ImageRequest contains all parameters needed for an image generation request.
type ImageRequest struct {
	Prompt     string
	Resolution string // "2K", "1080p", "720p"
	Seed       string
	Quality    string
	Watermark  bool
	References []ImageReference
}

// ImageReference represents a media file used as reference for image generation.
type ImageReference struct {
	Type    string // "image", "video"
	DataURL string
}

// ImageOutput is a single generated image resource.
type ImageOutput struct {
	URL string
}

// ImageResult is the response from an image generation task.
type ImageResult struct {
	TaskID  string
	Model   string
	Status  string // "succeeded", "failed"
	Outputs []ImageOutput
	Error   string
	Raw     interface{}
}
