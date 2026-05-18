// Package audio defines the domain contracts for audio generation.
//
// AudioGenerator is the interface that all audio generators must implement.
package audio

import "dcs-back-v0/internal/studio"

// AudioGenerator defines the contract that all audio generators must satisfy.
type AudioGenerator interface {
	Name() string
	Match(modelName string) bool
	Validate(req *studio.GeneratorRequest) error
	Generate(req *studio.GeneratorRequest) (*studio.GeneratorResult, error)
	GetStatus(taskID, apiKey, baseURL, endpoint string) (*studio.GeneratorResult, error)
	CancelTask(taskID, apiKey, baseURL, endpoint string) error
	BuildPayload(req *studio.GeneratorRequest) map[string]interface{}
	ContentType() string
}

// AudioRequest contains parameters for an audio generation request.
type AudioRequest struct {
	Prompt     string
	Duration   int
	Seed       string
	References []AudioReference
}

// AudioReference represents a media file used as reference.
type AudioReference struct {
	Type    string
	DataURL string
}

// AudioOutput is a single generated audio resource.
type AudioOutput struct {
	URL      string
	LocalURL string
}

// AudioResult is the response from an audio generation task.
type AudioResult struct {
	TaskID  string
	Model   string
	Status  string
	Outputs []AudioOutput
	Error   string
	Raw     interface{}
}
