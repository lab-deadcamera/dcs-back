// Package text defines the domain contracts for text generation.
//
// TextGenerator is the interface that all text generators must implement.
package text

import "dcs-back-v0/internal/studio"

// TextGenerator defines the contract that all text generators must satisfy.
type TextGenerator interface {
	Name() string
	Match(modelName string) bool
	Validate(req *studio.GeneratorRequest) error
	Generate(req *studio.GeneratorRequest) (*studio.GeneratorResult, error)
	GetStatus(taskID, apiKey, baseURL, endpoint string) (*studio.GeneratorResult, error)
	CancelTask(taskID, apiKey, baseURL, endpoint string) error
	BuildPayload(req *studio.GeneratorRequest) map[string]interface{}
	ContentType() string
}

// TextRequest contains parameters for a text generation request.
type TextRequest struct {
	System string
	Prompt string
	Seed   string
	Params map[string]interface{}
}

// TextOutput is a single generated text resource.
type TextOutput struct {
	Text string
}

// TextResult is the response from a text generation task.
type TextResult struct {
	TaskID  string
	Model   string
	Status  string
	Outputs []TextOutput
	Error   string
	Raw     interface{}
}
