package generators

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

	"dcs-back-v0/internal/studio"
)

var nameModeldreamina = "dreamina-seedance-2-0-260128"

// ─── Model-specific validation ─────────────────────────────────

// ValidRatios lists supported aspect ratios for video generation.
var ValidRatios = map[string]bool{
	"16:9": true, "9:16": true, "1:1": true, "4:3": true, "3:4": true,
	"21:9": true, "9:21": true, "2.35:1": true,
}

// ValidResolutionsVideo lists supported video resolutions.
var ValidResolutionsVideo = map[string]bool{
	"480p": true, "720p": true, "1080p": true,
}

// IsFastModel returns true if the model name contains "fast".
func IsFastModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "fast")
}

// VideoURLPattern matches HTTP(S) URLs for video file discovery.
var VideoURLPattern = regexp.MustCompile(`^https?://`)

// SafeSuffix returns the last 8 characters of a task ID for use in filenames.
func SafeSuffix(taskID string) string {
	if len(taskID) >= 8 {
		return taskID[len(taskID)-8:]
	}
	return taskID
}

// ─── SeedanceGenerator ─────────────────────────────────────────

type SeedanceGenerator struct {
	httpClient *http.Client
	outputsDir string
}

func NewSeedanceGenerator(outputsDir string) *SeedanceGenerator {
	return &SeedanceGenerator{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		outputsDir: outputsDir,
	}
}

func (g *SeedanceGenerator) Name() string { return nameModeldreamina }

func (g *SeedanceGenerator) ContentType() string { return "video" }

func (g *SeedanceGenerator) Match(modelName string) bool {
	lower := strings.ToLower(modelName)
	return strings.Contains(lower, nameModeldreamina)
}

func (g *SeedanceGenerator) Validate(req *studio.GeneratorRequest) error {
	errs := studio.ValidateCommon(req)
	if errs.HasErrors() {
		return errs
	}

	if req.Duration < 1 || req.Duration > 60 {
		errs.Add("duration", "must be between 1 and 60 seconds")
	}
	if req.Ratio != "" && !ValidRatios[req.Ratio] {
		errs.Add("ratio", "unsupported value: "+req.Ratio)
	}
	if req.Resolution != "" && !ValidResolutionsVideo[req.Resolution] {
		errs.Add("resolution", "must be one of: 480p, 720p, 1080p")
	}
	if req.GenerateAudio && IsFastModel(req.Model) {
		errs.Add("generate_audio", "only supported on pro models (non-fast)")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

func (g *SeedanceGenerator) Generate(req *studio.GeneratorRequest) (*studio.GeneratorResult, error) {
	payload := g.BuildPayload(req)

	result, err := g.arkRequest(req.BaseURL+req.Endpoint, "POST", payload, req.APIKey)
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

	return &studio.GeneratorResult{
		TaskID:  taskID,
		Model:   req.Model,
		Status:  "running",
		Outputs: []studio.OutputResource{},
		Raw:     result,
	}, nil
}

func (g *SeedanceGenerator) GetStatus(taskID, apiKey, baseURL, endpoint string) (*studio.GeneratorResult, error) {
	result, err := g.arkRequest(baseURL+endpoint+"/"+taskID, "GET", nil, apiKey)
	if err != nil {
		return nil, err
	}

	status, _ := result["status"].(string)

	if status == "succeeded" {
		videoURL := g.findVideoURL(result, 0)
		if videoURL != "" {
			localName := fmt.Sprintf("seedance_%d_%s.mp4", time.Now().UnixMilli(), SafeSuffix(taskID))
			localPath := filepath.Join(g.outputsDir, localName)

			outputs := []studio.OutputResource{{
				URL:  videoURL,
				Type: "video",
			}}

			vidResp, vidErr := http.Get(videoURL)
			if vidErr == nil {
				defer vidResp.Body.Close()
				vidBytes, readErr := io.ReadAll(vidResp.Body)
				if readErr == nil {
					os.WriteFile(localPath, vidBytes, 0644)
					outputs[0].LocalURL = "/outputs/" + localName
				}
			}

			return &studio.GeneratorResult{
				TaskID:  taskID,
				Model:   nameModeldreamina,
				Status:  status,
				Outputs: outputs,
				Raw:     result,
			}, nil
		}

		return &studio.GeneratorResult{
			TaskID:  taskID,
			Model:   nameModeldreamina,
			Status:  "succeeded_no_url",
			Outputs: []studio.OutputResource{},
			Raw:     result,
			Error:   "Job succeeded but no video URL was found in the response.",
		}, nil
	}

	if status == "failed" {
		errorMsg, _ := result["error"].(string)
		if errorMsg == "" {
			if e, ok := result["error"].(map[string]interface{}); ok {
				errorMsg, _ = e["message"].(string)
			}
		}
		return &studio.GeneratorResult{
			TaskID:  taskID,
			Model:   nameModeldreamina,
			Status:  status,
			Outputs: []studio.OutputResource{},
			Raw:     result,
			Error:   errorMsg,
		}, nil
	}

	return &studio.GeneratorResult{
		TaskID:  taskID,
		Model:   nameModeldreamina,
		Status:  status,
		Outputs: []studio.OutputResource{},
		Raw:     result,
	}, nil
}

func (g *SeedanceGenerator) CancelTask(taskID, apiKey, baseURL, endpoint string) error {
	_, err := g.arkRequest(baseURL+endpoint+"/"+taskID, "DELETE", nil, apiKey)
	return err
}

func (g *SeedanceGenerator) BuildPayload(req *studio.GeneratorRequest) map[string]interface{} {
	content := make([]map[string]interface{}, 0)
	imageIndex := 0
	videoIndex := 0
	audioIndex := 0

	for _, item := range req.Content {
		switch item.Type {
		case "image":
			if item.DataURL == "" {
				continue
			}
			content = append(content, map[string]interface{}{
				"type":      "image_url",
				"image_url": map[string]string{"url": item.DataURL},
				"role":      "reference_image",
			})
			imageIndex++
		case "video":
			if item.DataURL == "" {
				continue
			}
			content = append(content, map[string]interface{}{
				"type":      "video_url",
				"video_url": map[string]string{"url": item.DataURL},
				"role":      "reference_video",
			})
			videoIndex++
		case "audio":
			if item.DataURL == "" {
				continue
			}
			content = append(content, map[string]interface{}{
				"type":      "audio_url",
				"audio_url": map[string]string{"url": item.DataURL},
				"role":      "reference_audio",
			})
			audioIndex++
		}
	}

	textPart := studio.CompileContentText(req.Content)
	if imageIndex > 0 || videoIndex > 0 {
		refs := []string{}
		if imageIndex > 0 {
			refs = append(refs, "Image 1")
		}
		if videoIndex > 0 {
			refs = append(refs, "Video 1")
		}
		if len(refs) > 0 {
			textPart = fmt.Sprintf("The video references %s. ", strings.Join(refs, " and ")) + textPart
		}
	}

	content = append(content, map[string]interface{}{
		"type": "text",
		"text": textPart,
	})

	duration := req.Duration
	if duration <= 0 {
		duration = 5
	}

	payload := map[string]interface{}{
		"model":          req.Model,
		"content":        content,
		"ratio":          req.Ratio,
		"duration":       duration,
		"camerafixed":    req.CameraFixed,
		"watermark":      req.Watermark,
		"generate_audio": req.GenerateAudio,
	}

	if req.Ratio != "" {
		payload["ratio"] = req.Ratio
	}
	if req.Resolution != "" {
		payload["resolution"] = req.Resolution
	}
	if !req.GenerateAudio {
		payload["generate_audio"] = false
	}

	return payload
}

func (g *SeedanceGenerator) arkRequest(url, method string, body interface{}, apiKey string) (map[string]interface{}, error) {
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

	resp, err := g.httpClient.Do(req)
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
		return nil, fmt.Errorf("%s: %s", nameModeldreamina, string(respBytes))
	}

	if resp.StatusCode >= 400 {
		msg := studio.ExtractError(result, string(respBytes))
		return nil, fmt.Errorf("%s %d: %s", nameModeldreamina, resp.StatusCode, msg)
	}

	return result, nil
}

func (g *SeedanceGenerator) findVideoURL(obj interface{}, depth int) string {
	if obj == nil || depth > 6 {
		return ""
	}
	switch v := obj.(type) {
	case string:
		if VideoURLPattern.MatchString(v) {
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
			if found := g.findVideoURL(item, depth+1); found != "" {
				return found
			}
		}
		return ""
	case map[string]interface{}:
		for _, k := range []string{"video_url", "videoUrl", "url", "video"} {
			if s, ok := v[k].(string); ok && VideoURLPattern.MatchString(s) {
				return s
			}
		}
		for _, val := range v {
			if found := g.findVideoURL(val, depth+1); found != "" {
				return found
			}
		}
		return ""
	}
	return ""
}
