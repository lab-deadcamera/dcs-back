package generators

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dcs-back-v0/internal/modules/studio"
	"dcs-back-v0/internal/utils"
)

type GeminiNanoGenerator struct {
	httpClient *http.Client
	outputsDir string
}

func NewGeminiNanoGenerator(outputsDir string) *GeminiNanoGenerator {
	return &GeminiNanoGenerator{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		outputsDir: outputsDir,
	}
}

func (g *GeminiNanoGenerator) Name() string { return "gemini-nano-banana" }

func (g *GeminiNanoGenerator) ContentType() string { return "image" }

func (g *GeminiNanoGenerator) Match(modelName string) bool {
	lower := strings.ToLower(modelName)
	return strings.Contains(lower, "gemini")
}

func (g *GeminiNanoGenerator) Validate(req *studio.GeneratorRequest) error {
	errs := studio.ValidateCommon(req)
	if errs.HasErrors() {
		return errs
	}

	if req.Resolution != "" && !ValidResolutionsImage[req.Resolution] {
		errs.Add("resolution", "must be one of: 2K, 1080p, 720p")
	}
	if req.Duration > 0 {
		errs.Add("duration", "not supported for image generation")
	}
	if req.CameraFixed {
		errs.Add("camerafixed", "not supported for image generation")
	}
	if req.GenerateAudio {
		errs.Add("generate_audio", "not supported for image generation")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

func (g *GeminiNanoGenerator) BuildPayload(req *studio.GeneratorRequest) map[string]interface{} {
	textPart := studio.CompileContentText(req.Content)

	parts := []map[string]interface{}{
		{"text": textPart},
	}

	// Add reference images from content items
	for _, item := range req.Content {
		if item.Type != "image" || item.DataURL == "" {
			continue
		}
		imgPart := g.buildImagePart(item.DataURL)
		if imgPart != nil {
			parts = append(parts, imgPart)
		}
	}

	contents := []map[string]interface{}{
		{
			"parts": parts,
		},
	}

	payload := map[string]interface{}{
		"contents": contents,
	}

	genConfig := map[string]interface{}{}
	modalities := []string{"TEXT", "IMAGE"}
	genConfig["responseModalities"] = modalities

	genConfig["responseFormat"] = map[string]interface{}{
		"image": map[string]interface{}{
			"imageSize":   geminiImageSize(req.Resolution),
			"aspectRatio": geminiAspectRatio(req.Ratio),
		},
	}

	payload["generationConfig"] = genConfig
	return payload
}

// geminiImageSize maps human-friendly resolution strings to Gemini API enum values.
func geminiImageSize(resolution string) string {
	switch strings.ToUpper(resolution) {
	case "720P", "720PX", "1080P", "1K":
		return "IMAGE_SIZE_1K"
	case "2K":
		return "IMAGE_SIZE_2K"
	case "4K":
		return "IMAGE_SIZE_4K"
	default:
		return "IMAGE_SIZE_1K"
	}
}

// geminiAspectRatio maps "W:H" strings to Gemini API enum values.
func geminiAspectRatio(ratio string) string {
	switch ratio {
	case "1:1":
		return "ASPECT_RATIO_1_1"
	case "3:4":
		return "ASPECT_RATIO_3_4"
	case "4:3":
		return "ASPECT_RATIO_4_3"
	case "9:16":
		return "ASPECT_RATIO_9_16"
	case "16:9":
		return "ASPECT_RATIO_16_9"
	default:
		return "ASPECT_RATIO_1_1"
	}
}

// buildImagePart converts a DataURL to a Gemini API part.
// Supports both base64 data URLs and external HTTP(S) URLs.
func (g *GeminiNanoGenerator) buildImagePart(dataURL string) map[string]interface{} {
	if strings.HasPrefix(dataURL, "data:") {
		// data:{mimeType};base64,{data}
		commaIdx := strings.Index(dataURL, ",")
		if commaIdx < 0 {
			return nil
		}
		header := dataURL[:commaIdx]
		encoded := dataURL[commaIdx+1:]

		mimeType := "image/png"
		if strings.HasPrefix(header, "data:") {
			parts := strings.Split(header[5:], ";")
			if len(parts) > 0 && parts[0] != "" {
				mimeType = parts[0]
			}
		}

		return map[string]interface{}{
			"inline_data": map[string]interface{}{
				"mime_type": mimeType,
				"data":      encoded,
			},
		}
	}

	// External URL
	if strings.HasPrefix(dataURL, "http://") || strings.HasPrefix(dataURL, "https://") {
		ext := strings.ToLower(filepath.Ext(dataURL))
		mimeType := "image/png"
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".webp":
			mimeType = "image/webp"
		case ".gif":
			mimeType = "image/gif"
		}
		data, _ := utils.DownloadFromURL(dataURL)
		return map[string]interface{}{
			"inline_data": map[string]interface{}{
				"data":      base64.StdEncoding.EncodeToString(data),
				"mime_type": mimeType,
			},
		}
	}

	return nil
}

func (g *GeminiNanoGenerator) Generate(req *studio.GeneratorRequest) (*studio.GeneratorResult, error) {
	payload := g.BuildPayload(req)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	apiURL := req.BaseURL + req.Endpoint
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("x-goog-api-key", req.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gemini response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gemini API %d: %s", resp.StatusCode, string(respBytes))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse gemini response: %s", string(respBytes))
	}

	taskID := fmt.Sprintf("gemini_%d", time.Now().UnixMilli())
	var outputs []studio.OutputResource

	candidates, _ := result["candidates"].([]interface{})
	if len(candidates) > 0 {
		cand, _ := candidates[0].(map[string]interface{})
		content, _ := cand["content"].(map[string]interface{})
		parts, _ := content["parts"].([]interface{})

		for i, part := range parts {
			p, _ := part.(map[string]interface{})
			inlineData, _ := p["inlineData"].(map[string]interface{})
			if inlineData == nil {
				inlineData, _ = p["inline_data"].(map[string]interface{})
			}
			if inlineData == nil {
				continue
			}

			mimeType, _ := inlineData["mimeType"].(string)
			if mimeType == "" {
				mimeType, _ = inlineData["mime_type"].(string)
			}
			data, _ := inlineData["data"].(string)
			if data == "" {
				continue
			}

			imageBytes, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				continue
			}

			ext := ".png"
			if strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg") {
				ext = ".jpg"
			} else if strings.Contains(mimeType, "webp") {
				ext = ".webp"
			}

			outputFilename := fmt.Sprintf("gemini_%s_%d%s", taskID[7:], i, ext)
			outputPath := filepath.Join(g.outputsDir, outputFilename)

			if err := os.MkdirAll(g.outputsDir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create outputs dir: %w", err)
			}

			if err := os.WriteFile(outputPath, imageBytes, 0644); err != nil {
				return nil, fmt.Errorf("failed to write image: %w", err)
			}

			outputs = append(outputs, studio.OutputResource{
				URL:  "/outputs/" + outputFilename,
				Type: "image",
			})
		}
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("gemini: no image data found in response")
	}

	return &studio.GeneratorResult{
		TaskID:  taskID,
		Model:   req.Model,
		Status:  "succeeded",
		Outputs: outputs,
		Raw:     result,
	}, nil
}

func (g *GeminiNanoGenerator) GetStatus(taskID, apiKey, baseURL, endpoint string) (*studio.GeneratorResult, error) {
	return &studio.GeneratorResult{
		TaskID:  taskID,
		Status:  "succeeded",
		Outputs: []studio.OutputResource{},
	}, nil
}

func (g *GeminiNanoGenerator) CancelTask(taskID, apiKey, baseURL, endpoint string) error {
	return nil
}
