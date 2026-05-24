package studio

import (
	"fmt"
	"strings"
	"time"
)

type SeedreamHandler struct {
	seedance *SeedanceHandler
}

func NewSeedreamHandler(outputsDir string) *SeedreamHandler {
	return &SeedreamHandler{
		seedance: NewSeedanceHandler(outputsDir),
	}
}

func (h *SeedreamHandler) Matches(modelName string) bool {
	return strings.Contains(strings.ToLower(modelName), "seedream")
}

func (h *SeedreamHandler) Generate(sel *Selection, apiKey, baseURL, endpoint string) (*GenerateResponse, error) {
	payload := map[string]interface{}{
		"model":           sel.ModelID,
		"prompt":          strings.TrimSpace(sel.UserPrompt),
		"size":            "2K",
		"response_format": "url",
		"watermark":       false,
	}

	if sel.Resolution != nil && sel.Resolution.Value != "" {
		payload["size"] = sel.Resolution.Value
	}

	if len(sel.RefImages) > 0 {
		urls := make([]string, 0, len(sel.RefImages))
		for _, img := range sel.RefImages {
			if img.DataUrl != "" {
				urls = append(urls, img.DataUrl)
			}
		}
		if len(urls) == 1 {
			payload["image"] = urls[0]
		} else if len(urls) > 1 {
			payload["image"] = urls
		}
	}

	result, err := h.seedance.arkRequest("seedream", baseURL+endpoint+"/images/generations", "POST", payload, apiKey)
	if err != nil {
		return nil, err
	}

	taskID, _ := result["id"].(string)
	if taskID == "" {
		taskID = fmt.Sprintf("seedream_%d", time.Now().UnixMilli())
	}

	return &GenerateResponse{
		TaskID:  taskID,
		ModelID: sel.ModelID,
		Prompt:  sel.UserPrompt,
		Model:   "seedream",
		Status:  "succeeded",
	}, nil
}

func (h *SeedreamHandler) GetStatus(taskID, apiKey, baseURL, endpoint string) (*StatusResult, error) {
	return nil, fmt.Errorf("seedream tasks complete synchronously, use the generate response directly")
}

func (h *SeedreamHandler) CancelTask(taskID, apiKey, baseURL, endpoint string) error {
	return fmt.Errorf("seedream tasks cannot be cancelled")
}
