// Package video defines the domain contracts for video generation.
//
// VideoGenerator is the interface that all video generators must implement.
// VideoRequest/VideoResult define the data it receives and produces at the domain level.
// Internally, generators work with shared types from the generators package.
package video

import "dcs-back-v0/internal/studio"

// VideoGenerator defines the contract that all video generators must satisfy.
type VideoGenerator interface {
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
	// ContentType returns "video".
	ContentType() string
}

// VideoRequest contains all parameters needed for a video generation request.
type VideoRequest struct {
	Prompt     string
	Duration   int    // seconds (1–60)
	Ratio      string // aspect ratio (e.g. "16:9", "9:16", "1:1")
	Resolution string // "480p", "720p", "1080p"
	CameraFix  bool
	Seed       string
	Quality    string
	Watermark  bool
	Audio      bool // generate audio track
	References []VideoReference
}

// VideoReference represents a media file used as reference for generation.
type VideoReference struct {
	Type    string // "image", "video", "audio"
	DataURL string
}

// VideoOutput is a single generated video resource.
type VideoOutput struct {
	URL      string
	LocalURL string
}

// VideoResult is the response from a video generation task.
type VideoResult struct {
	TaskID  string
	Model   string
	Status  string // "running", "succeeded", "failed"
	Outputs []VideoOutput
	Error   string
	Raw     interface{}
}
