package generators

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"dcs-back-v0/internal/modules/studio"
)

const (
	claudeModelName = "claude-shot-builder"
)

// ClaudeTextGenerator implements the text.TextGenerator interface for Anthropic Claude.
type ClaudeTextGenerator struct {
	httpClient *http.Client
}

// NewClaudeTextGenerator creates a new Claude text generator instance.
func NewClaudeTextGenerator() *ClaudeTextGenerator {
	return &ClaudeTextGenerator{
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Name returns the generator model identifier.
func (g *ClaudeTextGenerator) Name() string { return claudeModelName }

// ContentType returns the media type produced by this generator.
func (g *ClaudeTextGenerator) ContentType() string { return "text" }

// Match determines whether this generator handles the requested model name.
func (g *ClaudeTextGenerator) Match(modelName string) bool {
	lower := modelName
	return lower == claudeModelName || lower == "claude"
}

// Ensure validates the request against the generator capabilities.
func (g *ClaudeTextGenerator) Validate(req *studio.GeneratorRequest) error {
	errs := studio.ValidateCommon(req)
	if errs.HasErrors() {
		return errs
	}

	if req.Resolution != "" {
		errs.Add("resolution", "text generation does not support resolution overrides")
	}
	if req.Duration > 0 {
		errs.Add("duration", "text generation does not support duration")
	}
	if req.CameraFixed {
		errs.Add("camerafixed", "text generation does not support camera fixed")
	}
	if req.GenerateAudio {
		errs.Add("generate_audio", "text generation does not support audio generation")
	}
	if req.Watermark {
		errs.Add("watermark", "text generation does not support watermark")
	}
	if len(req.Content) == 0 {
		errs.Add("content", "text generation requires at least one content item")
	}

	if errs.HasErrors() {
		return errs
	}

	return nil
}

// BuildPayload converts the studio request into the Claude API payload.
func (g *ClaudeTextGenerator) BuildPayload(req *studio.GeneratorRequest) map[string]interface{} {
	messages := g.buildMessages(req)
	if len(messages) == 0 {
		messages = []map[string]interface{}{
			{"role": "user", "content": studio.CompileContentText(req.Content)},
		}
	}

	// Map internal model name to a real Anthropic model ID.
	apiModel := req.Model
	switch req.Model {
	case "claude-shot-builder", "claude":
		apiModel = "claude-3-haiku-20240307"
	}

	return map[string]interface{}{
		"model":      apiModel,
		"max_tokens": 4096,
		"messages":   messages,
	}
}

// Generate sends the request to the Claude API and returns the first text result.
func (g *ClaudeTextGenerator) Generate(req *studio.GeneratorRequest) (*studio.GeneratorResult, error) {
	payload := g.BuildPayload(req)

	result, err := g.claudeRequest(req.BaseURL+req.Endpoint, "POST", payload, req.APIKey)
	if err != nil {
		return nil, err
	}

	taskID := g.extractTaskID(result)
	text := g.extractRawText(result)

	return &studio.GeneratorResult{
		TaskID:  taskID,
		Model:   req.Model,
		Status:  "succeeded",
		Outputs: []studio.OutputResource{g.newTextOutput(text)},
		Raw:     result,
	}, nil
}

// GetStatus polls the Claude API for completion state.
func (g *ClaudeTextGenerator) GetStatus(taskID, apiKey, baseURL, endpoint string) (*studio.GeneratorResult, error) {
	url := fmt.Sprintf("%s%s/messages/%s", baseURL, endpoint, taskID)
	result, err := g.claudeRequest(url, "GET", nil, apiKey)
	if err != nil {
		return nil, err
	}

	return &studio.GeneratorResult{
		TaskID:  taskID,
		Model:   claudeModelName,
		Status:  g.mapStatus(result),
		Outputs: []studio.OutputResource{g.newTextOutput(g.extractRawText(result))},
		Raw:     result,
	}, nil
}

// CancelTask is a no-op for synchronous text generators.
func (g *ClaudeTextGenerator) CancelTask(taskID, apiKey, baseURL, endpoint string) error {
	return nil
}

func (g *ClaudeTextGenerator) buildMessages(req *studio.GeneratorRequest) []map[string]interface{} {
	var msgs []map[string]interface{}
	foundSystem := false

	for _, item := range req.Content {
		if item.Type != "text" {
			continue
		}

		trimmed := ""
		if item.Text != "" {
			trimmed = item.Text
		}

		// Heuristic: if the prompt starts with a "system" marker, treat it as system prompt.
		if !foundSystem && len(trimmed) > 7 && trimmed[:7] == "[SYSTEM" {
			msgs = append(msgs, map[string]interface{}{"role": "system", "content": trimmed})
			foundSystem = true
			continue
		}

		msgs = append(msgs, map[string]interface{}{"role": "user", "content": trimmed})
	}

	return msgs
}

// claudeRequest sends an HTTP request to the Anthropic API.
func (g *ClaudeTextGenerator) claudeRequest(url, method string, body interface{}, apiKey string) (map[string]interface{}, error) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Claude payload: %w", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create Claude request: %w", err)
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Claude request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Claude response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("Claude response parse error: %s", string(respBytes))
	}

	if resp.StatusCode >= 400 {
		msg := studio.ExtractError(result, string(respBytes))
		fmt.Printf("[claude-debug] url=%q status=%d apiKey=%q body=%s\n", url, resp.StatusCode, safeKeyPrefix(apiKey), string(respBytes))
		return nil, fmt.Errorf("Claude API error %d: %s", resp.StatusCode, msg)
	}

	return result, nil
}

// extractTaskID extracts a message ID from the Claude response.
func (g *ClaudeTextGenerator) extractTaskID(result map[string]interface{}) string {
	if id, ok := result["id"].(string); ok && id != "" {
		return id
	}
	return fmt.Sprintf("claude_text_%d", time.Now().UnixMilli())
}

// extractRawText extracts the raw text content from the Claude response body.
func (g *ClaudeTextGenerator) extractRawText(result map[string]interface{}) string {
	content, ok := result["content"].([]interface{})
	if !ok {
		return ""
	}

	for _, block := range content {
		if blockMap, ok := block.(map[string]interface{}); ok {
			if blockMap["type"] == "text" {
				if text, ok := blockMap["text"].(string); ok {
					return text
				}
			}
		}
	}

	return ""
}

// mapStatus maps the Claude response to a canonical status string.
func (g *ClaudeTextGenerator) mapStatus(result map[string]interface{}) string {
	if stopReason, ok := result["stop_reason"].(string); ok && stopReason != "" {
		switch stopReason {
		case "end_turn", "stop_sequence", "max_tokens":
			return "succeeded"
		case "error":
			return "failed"
		}
	}
	return "succeeded"
}

// newTextOutput creates a text output resource using a base64 data URL hack over the existing URL field.
func (g *ClaudeTextGenerator) newTextOutput(text string) studio.OutputResource {
	if text == "" {
		return studio.OutputResource{}
	}

	return studio.OutputResource{
		URL:  fmt.Sprintf("data:text/plain;base64,%s", base64.StdEncoding.EncodeToString([]byte(text))),
		Type: "text",
	}
}

// safeKeyPrefix returns the first 10 chars of an API key for debugging, or "empty" if blank.
func safeKeyPrefix(key string) string {
	if len(key) > 10 {
		return key[:10] + "..."
	}
	if key == "" {
		return "empty"
	}
	return key
}
