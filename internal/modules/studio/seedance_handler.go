package studio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var videoURLPattern = regexp.MustCompile(`^https?://`)

type SeedanceHandler struct {
	httpClient *http.Client
	outputsDir string
}

func NewSeedanceHandler(outputsDir string) *SeedanceHandler {
	return &SeedanceHandler{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		outputsDir: outputsDir,
	}
}

func (h *SeedanceHandler) Matches(modelName string) bool {
	lower := strings.ToLower(modelName)
	return strings.Contains(lower, "seedance") || strings.Contains(lower, "dreamina")
}

func (h *SeedanceHandler) Generate(sel *Selection, apiKey, baseURL, endpoint string) (*GenerateResponse, error) {
	modelName, _ := h.resolveModelName(baseURL, endpoint)
	payload, compiledText := h.buildPayload(sel, modelName)

	result, err := h.arkRequest(modelName, baseURL+endpoint+"/contents/generations/tasks", "POST", payload, apiKey)
	if err != nil {
		return nil, err
	}

	taskID, _ := result["id"].(string)
	if taskID == "" {
		taskID, _ = result["task_id"].(string)
	}
	if taskID == "" {
		return nil, fmt.Errorf("no task ID in response")
	}

	return &GenerateResponse{
		TaskID:  taskID,
		ModelID: sel.ModelID,
		Prompt:  compiledText,
		Model:   modelName,
		Status:  "running",
	}, nil
}

func (h *SeedanceHandler) GetStatus(taskID, apiKey, baseURL, endpoint string) (*StatusResult, error) {
	modelName, _ := h.resolveModelName(baseURL, endpoint)
	result, err := h.arkRequest(modelName, baseURL+endpoint+"/contents/generations/tasks/"+taskID, "GET", nil, apiKey)
	if err != nil {
		return nil, err
	}

	status, _ := result["status"].(string)

	if status == "succeeded" {
		videoURL := h.findVideoURL(result, 0)
		if videoURL != "" {
			localName := fmt.Sprintf("seedance_%d_%s.mp4", time.Now().UnixMilli(), safeSuffix(taskID))
			localPath := filepath.Join(h.outputsDir, localName)

			vidResp, vidErr := http.Get(videoURL)
			if vidErr == nil {
				defer vidResp.Body.Close()
				vidBytes, readErr := io.ReadAll(vidResp.Body)
				if readErr == nil {
					os.WriteFile(localPath, vidBytes, 0644)
					return &StatusResult{
						Status:   status,
						VideoURL: videoURL,
						LocalURL: "/outputs/" + localName,
						Raw:      result,
					}, nil
				}
			}
			return &StatusResult{
				Status:   status,
				VideoURL: videoURL,
				LocalURL: "",
				Raw:      result,
			}, nil
		}
		return &StatusResult{
			Status: "succeeded_no_url",
			Raw:    result,
			Error:  "Job succeeded but no video URL was found in the response.",
		}, nil
	}

	if status == "failed" {
		errorMsg, _ := result["error"].(string)
		if errorMsg == "" {
			if e, ok := result["error"].(map[string]interface{}); ok {
				errorMsg, _ = e["message"].(string)
			}
		}
		return &StatusResult{
			Status: status,
			Error:  errorMsg,
			Raw:    result,
		}, nil
	}

	return &StatusResult{
		Status: status,
		Raw:    result,
	}, nil
}

func (h *SeedanceHandler) CancelTask(taskID, apiKey, baseURL, endpoint string) error {
	modelName, _ := h.resolveModelName(baseURL, endpoint)
	_, err := h.arkRequest(modelName, baseURL+endpoint+"/contents/generations/tasks/"+taskID, "DELETE", nil, apiKey)
	return err
}

func (h *SeedanceHandler) resolveModelName(baseURL, endpoint string) (string, string) {
	return "seedance", baseURL + endpoint
}

func (h *SeedanceHandler) buildPayload(sel *Selection, model string) (map[string]interface{}, string) {
	content := make([]map[string]interface{}, 0)

	if sel.FirstFrame != nil && sel.FirstFrame.DataUrl != "" {
		content = append(content, map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]string{"url": sel.FirstFrame.DataUrl},
			"role":      "reference_image",
		})
	}
	if sel.LastFrame != nil && sel.LastFrame.DataUrl != "" {
		content = append(content, map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]string{"url": sel.LastFrame.DataUrl},
			"role":      "reference_image",
		})
	}
	for _, img := range sel.RefImages {
		if img.DataUrl != "" {
			content = append(content, map[string]interface{}{
				"type":      "image_url",
				"image_url": map[string]string{"url": img.DataUrl},
				"role":      "reference_image",
			})
		}
	}
	for _, vid := range sel.RefVideos {
		if vid.DataUrl != "" {
			content = append(content, map[string]interface{}{
				"type":      "video_url",
				"video_url": map[string]string{"url": vid.DataUrl},
				"role":      "reference_video",
			})
		}
	}
	for _, aud := range sel.RefAudios {
		if aud.DataUrl != "" {
			content = append(content, map[string]interface{}{
				"type":      "audio_url",
				"audio_url": map[string]string{"url": aud.DataUrl},
				"role":      "reference_audio",
			})
		}
	}

	textPart := compilePromptText(sel)
	var hints []string
	if sel.FirstFrame != nil && sel.FirstFrame.DataUrl != "" && sel.LastFrame != nil && sel.LastFrame.DataUrl != "" {
		hints = append(hints, "The video starts on Image 1 and ends on Image 2.")
	} else if sel.FirstFrame != nil && sel.FirstFrame.DataUrl != "" {
		hints = append(hints, "The video starts on Image 1.")
	}
	if len(hints) > 0 {
		textPart = strings.Join(hints, " ") + " " + textPart
	}
	content = append(content, map[string]interface{}{
		"type": "text",
		"text": textPart,
	})

	duration := int(sel.Duration)
	if duration <= 0 {
		duration = 5
	}

	payload := map[string]interface{}{
		"model":       model,
		"content":     content,
		"ratio":       "16:9",
		"duration":    duration,
		"camerafixed": false,
		"watermark":   false,
	}

	if sel.AspectRatio != nil && sel.AspectRatio.Value != "" {
		payload["ratio"] = sel.AspectRatio.Value
	}
	if sel.CameraMotion != nil && sel.CameraMotion.ID == "static_lockoff" {
		payload["camerafixed"] = true
	}
	if sel.SoundOn != nil {
		payload["generate_audio"] = *sel.SoundOn
	} else if !strings.Contains(strings.ToLower(model), "fast") {
		payload["generate_audio"] = true
	}

	return payload, textPart
}

func (h *SeedanceHandler) arkRequest(modelName, url, method string, body interface{}, apiKey string) (map[string]interface{}, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("%s: %s", modelName, string(respBytes))
	}

	if resp.StatusCode >= 400 {
		msg := extractError(result, string(respBytes))
		return nil, fmt.Errorf("%s %d: %s", modelName, resp.StatusCode, msg)
	}

	return result, nil
}

func (h *SeedanceHandler) findVideoURL(obj interface{}, depth int) string {
	if obj == nil || depth > 6 {
		return ""
	}
	switch v := obj.(type) {
	case string:
		if videoURLPattern.MatchString(v) {
			if strings.HasSuffix(strings.SplitN(v, "?", 2)[0], ".mp4") ||
				strings.Contains(v, "tos-") ||
				strings.Contains(v, "bytepluses.com") ||
				strings.Contains(v, "volces.com") ||
				strings.Contains(v, "byteimg.com") {
				return v
			}
		}
		return ""
	case []interface{}:
		for _, item := range v {
			if found := h.findVideoURL(item, depth+1); found != "" {
				return found
			}
		}
		return ""
	case map[string]interface{}:
		for _, k := range []string{"video_url", "videoUrl", "url", "video"} {
			if s, ok := v[k].(string); ok && videoURLPattern.MatchString(s) {
				return s
			}
		}
		for _, val := range v {
			if found := h.findVideoURL(val, depth+1); found != "" {
				return found
			}
		}
		return ""
	}
	return ""
}

func compilePromptText(sel *Selection) string {
	var parts []string
	if strings.TrimSpace(sel.UserPrompt) != "" {
		parts = append(parts, strings.TrimSpace(sel.UserPrompt))
	}
	if sel.Camera != nil && strings.TrimSpace(sel.Camera.Prompt) != "" {
		parts = append(parts, strings.TrimSpace(sel.Camera.Prompt))
	}
	if sel.Lens != nil && strings.TrimSpace(sel.Lens.Prompt) != "" {
		parts = append(parts, strings.TrimSpace(sel.Lens.Prompt))
	}
	if sel.CameraMotion != nil && strings.TrimSpace(sel.CameraMotion.Prompt) != "" {
		parts = append(parts, strings.TrimSpace(sel.CameraMotion.Prompt))
	}
	if sel.ColorGrading != nil && strings.TrimSpace(sel.ColorGrading.Prompt) != "" {
		parts = append(parts, strings.TrimSpace(sel.ColorGrading.Prompt))
	}
	if sel.Genre != nil && strings.TrimSpace(sel.Genre.Prompt) != "" {
		parts = append(parts, strings.TrimSpace(sel.Genre.Prompt))
	}
	textBlock := strings.Join(parts, ". ")
	if textBlock != "" && !strings.HasSuffix(textBlock, ".") {
		textBlock += "."
	}
	return textBlock
}

func safeSuffix(taskID string) string {
	if len(taskID) >= 8 {
		return taskID[len(taskID)-8:]
	}
	return taskID
}

func extractError(result map[string]interface{}, raw string) string {
	if e, ok := result["error"].(map[string]interface{}); ok {
		if msg, ok := e["message"].(string); ok {
			return msg
		}
	}
	if msg, ok := result["message"].(string); ok {
		return msg
	}
	if len(raw) > 400 {
		raw = raw[:400]
	}
	return raw
}
