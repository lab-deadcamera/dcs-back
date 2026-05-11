package studio

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var videoURLPattern = regexp.MustCompile(`^https?://`)

const (
	DefaultEndpointID = "byteplus_ap"
	DefaultModel      = "dreamina-seedance-2-0-fast-260128"
)

var Endpoints = map[string]Endpoint{
	"byteplus_ap": {
		Label: "BytePlus · Singapore (ap-southeast)",
		URL:   "https://ark.ap-southeast.bytepluses.com/api/v3",
	},
	"volcengine_cn": {
		Label: "Volcengine · China (cn-beijing)",
		URL:   "https://ark.cn-beijing.volces.com/api/v3",
	},
}

type activeKeyInfo struct {
	Value      string
	Endpoint   Endpoint
	EndpointID string
	AK         string
	SK         string
}

type trustedAsset struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Prompt    string `json:"prompt"`
	Model     string `json:"model"`
	Seed      int    `json:"seed,omitempty"`
	Size      string `json:"size"`
	CreatedAt int64  `json:"createdAt"`
	ExpiresAt int64  `json:"expiresAt"`
}

type Service struct {
	store           *Store
	httpClient      *http.Client
	trustedAssets   []trustedAsset
	loggedTasks     map[string]bool
	muAssets        sync.Mutex
}

func NewService(store *Store) *Service {
	return &Service{
		store:         store,
		httpClient:    &http.Client{Timeout: 60 * time.Second},
		loggedTasks:   make(map[string]bool),
	}
}

// ─── Key management ───────────────────────────────────────────

func (s *Service) getActiveKey() *activeKeyInfo {
	data, err := s.store.LoadKeys()
	if err != nil || data.Active == "" {
		return nil
	}
	for _, k := range data.Keys {
		if k.ID == data.Active {
			endpointID := k.Endpoint
			if endpointID == "" {
				endpointID = DefaultEndpointID
			}
			ep, ok := Endpoints[endpointID]
			if !ok {
				ep = Endpoints[DefaultEndpointID]
			}
			return &activeKeyInfo{
				Value:      k.Value,
				Endpoint:   ep,
				EndpointID: endpointID,
				AK:         k.AK,
				SK:         k.SK,
			}
		}
	}
	return nil
}

func (s *Service) generateID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func maskKey(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 12 {
		return "••••"
	}
	return v[:4] + "••••" + v[len(v)-4:]
}

func (s *Service) ListKeys() (*KeyListResponse, error) {
	data, err := s.store.LoadKeys()
	if err != nil {
		return nil, err
	}
	resp := &KeyListResponse{
		Active:    data.Active,
		Endpoints: Endpoints,
		Keys:      make([]MaskedKey, 0, len(data.Keys)),
	}
	for _, k := range data.Keys {
		resp.Keys = append(resp.Keys, MaskedKey{
			ID:        k.ID,
			Name:      k.Name,
			Preview:   maskKey(k.Value),
			Endpoint:  k.Endpoint,
			HasAkSk:   k.AK != "" && k.SK != "",
			AKPreview: maskKey(k.AK),
			CreatedAt: k.CreatedAt,
		})
	}
	return resp, nil
}

func (s *Service) AddKey(req AddKeyRequest) (*KeyListResponse, error) {
	data, err := s.store.LoadKeys()
	if err != nil {
		return nil, err
	}
	endpoint := DefaultEndpointID
	if _, ok := Endpoints[req.Endpoint]; ok {
		endpoint = req.Endpoint
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("key-%d", len(data.Keys)+1)
	}
	newKey := APIKey{
		ID:        s.generateID(),
		Name:      name,
		Value:     strings.TrimSpace(req.Value),
		Endpoint:  endpoint,
		AK:        strings.TrimSpace(req.AK),
		SK:        strings.TrimSpace(req.SK),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	data.Keys = append(data.Keys, newKey)
	if data.Active == "" {
		data.Active = newKey.ID
	}
	if err := s.store.SaveKeys(data); err != nil {
		return nil, err
	}
	return s.ListKeys()
}

func (s *Service) ActivateKey(id string) (*KeyListResponse, error) {
	data, err := s.store.LoadKeys()
	if err != nil {
		return nil, err
	}
	found := false
	for _, k := range data.Keys {
		if k.ID == id {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("key not found")
	}
	data.Active = id
	if err := s.store.SaveKeys(data); err != nil {
		return nil, err
	}
	return s.ListKeys()
}

func (s *Service) DeleteKey(id string) (*KeyListResponse, error) {
	data, err := s.store.LoadKeys()
	if err != nil {
		return nil, err
	}
	newKeys := make([]APIKey, 0, len(data.Keys))
	for _, k := range data.Keys {
		if k.ID != id {
			newKeys = append(newKeys, k)
		}
	}
	if data.Active == id {
		if len(newKeys) > 0 {
			data.Active = newKeys[0].ID
		} else {
			data.Active = ""
		}
	}
	data.Keys = newKeys
	if err := s.store.SaveKeys(data); err != nil {
		return nil, err
	}
	return s.ListKeys()
}

func (s *Service) UpdateKey(id string, req UpdateKeyRequest) (*KeyListResponse, error) {
	data, err := s.store.LoadKeys()
	if err != nil {
		return nil, err
	}
	var found *APIKey
	for i := range data.Keys {
		if data.Keys[i].ID == id {
			found = &data.Keys[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("key not found")
	}
	if strings.TrimSpace(req.Name) != "" {
		found.Name = strings.TrimSpace(req.Name)
	}
	if req.Endpoint != "" {
		if _, ok := Endpoints[req.Endpoint]; ok {
			found.Endpoint = req.Endpoint
		}
	}
	if err := s.store.SaveKeys(data); err != nil {
		return nil, err
	}
	return s.ListKeys()
}

// ─── Presets ──────────────────────────────────────────────────

func (s *Service) GetPresets() ([]byte, error) {
	return s.store.LoadPresets()
}

// ─── Prompt compiler ──────────────────────────────────────────

func compilePromptText(sel Selection) string {
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

func buildPayload(sel Selection, model string) (map[string]interface{}, string) {
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
	if sel.Resolution != nil && sel.Resolution.Value != "" {
		payload["resolution"] = sel.Resolution.Value
	}
	if sel.CameraMotion != nil && sel.CameraMotion.ID == "static_lockoff" {
		payload["camerafixed"] = true
	}

	isPro := !strings.Contains(strings.ToLower(model), "fast")
	if isPro {
		generateAudio := true
		if sel.SoundOn != nil && !*sel.SoundOn {
			generateAudio = false
		}
		payload["generate_audio"] = generateAudio
	}

	return payload, textPart
}

func (s *Service) CompilePrompt(sel Selection) string {
	_, text := buildPayload(sel, "")
	return text
}

// ─── BytePlus API request (Bearer) ────────────────────────────

func (s *Service) arkRequest(apiPath, method string, body interface{}) (map[string]interface{}, error) {
	active := s.getActiveKey()
	if active == nil {
		return nil, fmt.Errorf("no active API key. Add one from the panel.")
	}

	url := active.Endpoint.URL + apiPath
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
	req.Header.Set("Authorization", "Bearer "+active.Value)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
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
		return nil, fmt.Errorf("%s: %s", active.EndpointID, string(respBytes))
	}

	if resp.StatusCode >= 400 {
		msg := extractError(result, string(respBytes))
		return nil, fmt.Errorf("%s %d: %s", active.EndpointID, resp.StatusCode, msg)
	}

	return result, nil
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

// ─── Seedance generation ──────────────────────────────────────

func (s *Service) Generate(sel Selection) (map[string]interface{}, error) {
	model := sel.Model
	if model == "" {
		model = DefaultModel
	}
	payload, compiledText := buildPayload(sel, model)

	log.Printf("[generate] model=%s ratio=%v duration=%v", model, payload["ratio"], payload["duration"])

	result, err := s.arkRequest("/contents/generations/tasks", "POST", payload)
	if err != nil {
		return nil, err
	}

	taskID, _ := result["id"].(string)
	if taskID == "" {
		taskID, _ = result["task_id"].(string)
	}
	if taskID == "" {
		return nil, fmt.Errorf("no task ID returned from BytePlus")
	}

	return map[string]interface{}{
		"taskId": taskID,
		"prompt": compiledText,
		"model":  model,
	}, nil
}

// ─── Status polling ───────────────────────────────────────────

func (s *Service) GetStatus(taskID string) (map[string]interface{}, error) {
	result, err := s.arkRequest("/contents/generations/tasks/"+taskID, "GET", nil)
	if err != nil {
		return nil, err
	}

	status, _ := result["status"].(string)
	videoURL := findVideoUrl(result, 0)

	if status == "succeeded" && videoURL != "" {
		localName := fmt.Sprintf("seedance_%d_%s.mp4", time.Now().UnixMilli(), safeSuffix(taskID))
		localPath := filepath.Join(s.store.OutputsDir(), localName)

		vidResp, err := http.Get(videoURL)
		if err == nil {
			defer vidResp.Body.Close()
			vidBytes, readErr := io.ReadAll(vidResp.Body)
			if readErr == nil {
				os.WriteFile(localPath, vidBytes, 0644)
				return map[string]interface{}{
					"status":   status,
					"videoUrl": videoURL,
					"localUrl": "/outputs/" + localName,
					"raw":      result,
				}, nil
			}
		}
		return map[string]interface{}{
			"status":   status,
			"videoUrl": videoURL,
			"localUrl": nil,
		}, nil
	}

	if status == "succeeded" && videoURL == "" {
		return map[string]interface{}{
			"status": "succeeded_no_url",
			"raw":    result,
			"error":  "Job succeeded but no video URL was found in the response.",
		}, nil
	}

	return map[string]interface{}{
		"status": status,
		"raw":    result,
	}, nil
}

func safeSuffix(taskID string) string {
	if len(taskID) >= 8 {
		return taskID[len(taskID)-8:]
	}
	return taskID
}

// ─── Cancel task ──────────────────────────────────────────────

func (s *Service) CancelTask(taskID string) (map[string]interface{}, error) {
	return s.arkRequest("/contents/generations/tasks/"+taskID, "DELETE", nil)
}

// ─── Seedream image generation ────────────────────────────────

func (s *Service) GenerateSeedream(req SeedreamRequest) (map[string]interface{}, error) {
	model := req.Model
	if model == "" {
		model = "seedream-4-0-250828"
	}
	size := req.Size
	if size == "" {
		size = "2K"
	}

	payload := map[string]interface{}{
		"model":           model,
		"prompt":          strings.TrimSpace(req.Prompt),
		"size":            size,
		"response_format": "url",
		"watermark":       false,
	}
	if req.Seed != nil {
		payload["seed"] = *req.Seed
	}
	if len(req.ReferenceImages) > 0 {
		if len(req.ReferenceImages) == 1 {
			payload["image"] = req.ReferenceImages[0]
		} else {
			payload["image"] = req.ReferenceImages
		}
	}

	result, err := s.arkRequest("/images/generations", "POST", payload)
	if err != nil {
		return nil, err
	}

	var url string
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		if item, ok := data[0].(map[string]interface{}); ok {
			url, _ = item["url"].(string)
		}
	}
	if url == "" {
		return nil, fmt.Errorf("Seedream returned no image URL")
	}

	asset := trustedAsset{
		ID:        s.generateID(),
		URL:       url,
		Prompt:    strings.TrimSpace(req.Prompt),
		Model:     model,
		Size:      size,
		CreatedAt: time.Now().UnixMilli(),
		ExpiresAt: time.Now().UnixMilli() + 30*24*60*60*1000,
	}
	if req.Seed != nil {
		asset.Seed = *req.Seed
	}

	s.muAssets.Lock()
	s.trustedAssets = append([]trustedAsset{asset}, s.trustedAssets...)
	if len(s.trustedAssets) > 50 {
		s.trustedAssets = s.trustedAssets[:50]
	}
	s.muAssets.Unlock()

	return map[string]interface{}{
		"id":        asset.ID,
		"url":       asset.URL,
		"prompt":    asset.Prompt,
		"model":     asset.Model,
		"seed":      asset.Seed,
		"size":      asset.Size,
		"createdAt": asset.CreatedAt,
		"expiresAt": asset.ExpiresAt,
		"raw":       result,
	}, nil
}

// ─── Trusted assets ───────────────────────────────────────────

func (s *Service) ListTrustedAssets() map[string]interface{} {
	s.muAssets.Lock()
	defer s.muAssets.Unlock()

	now := time.Now().UnixMilli()
	valid := make([]trustedAsset, 0, len(s.trustedAssets))
	for _, a := range s.trustedAssets {
		if a.ExpiresAt > now {
			valid = append(valid, a)
		}
	}
	s.trustedAssets = valid

	assets := make([]map[string]interface{}, len(valid))
	for i, a := range valid {
		assets[i] = map[string]interface{}{
			"id":        a.ID,
			"url":       a.URL,
			"prompt":    a.Prompt,
			"model":     a.Model,
			"seed":      a.Seed,
			"size":      a.Size,
			"createdAt": a.CreatedAt,
			"expiresAt": a.ExpiresAt,
		}
	}
	return map[string]interface{}{"assets": assets}
}

// ─── Video URL finder ─────────────────────────────────────────

func findVideoUrl(obj interface{}, depth int) string {
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
			if found := findVideoUrl(item, depth+1); found != "" {
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
			if found := findVideoUrl(val, depth+1); found != "" {
				return found
			}
		}
		return ""
	}
	return ""
}

// ─── Assets API wrappers ──────────────────────────────────────

func (s *Service) getActiveKeyChecked() (*activeKeyInfo, error) {
	active := s.getActiveKey()
	if active == nil {
		return nil, fmt.Errorf("no active API key")
	}
	if active.AK == "" || active.SK == "" {
		return nil, fmt.Errorf("active key has no AK/SK configured. Add them in the API panel")
	}
	return active, nil
}

func (s *Service) CreateAssetGroup(name, description, projectName string) (map[string]interface{}, error) {
	active, err := s.getActiveKeyChecked()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("group name required")
	}
	if projectName == "" {
		projectName = "default"
	}
	result, err := SignedFetch(SignedFetchInput{
		AK:      active.AK,
		SK:      active.SK,
		Region:  AssetsRegion,
		Service: AssetsService,
		Action:  "CreateAssetGroup",
		Version: AssetsVersion,
		Body: map[string]interface{}{
			"Name":        strings.TrimSpace(name),
			"Description": description,
			"GroupType":   "AIGC",
			"ProjectName": projectName,
		},
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":          result["Id"],
		"name":        name,
		"description": description,
		"projectName": projectName,
	}, nil
}

func (s *Service) ListAssetGroups() (map[string]interface{}, error) {
	active, err := s.getActiveKeyChecked()
	if err != nil {
		return nil, err
	}
	result, err := SignedFetch(SignedFetchInput{
		AK:      active.AK,
		SK:      active.SK,
		Region:  AssetsRegion,
		Service: AssetsService,
		Action:  "ListAssetGroups",
		Version: AssetsVersion,
		Body: map[string]interface{}{
			"Filter":     map[string]string{"GroupType": "AIGC"},
			"PageNumber": 1,
			"PageSize":   50,
		},
	})
	if err != nil {
		return nil, err
	}
	items, _ := result["Items"].([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	total, _ := result["TotalCount"].(float64)
	return map[string]interface{}{
		"groups": items,
		"total":  int(total),
	}, nil
}

func (s *Service) CreateAsset(groupID, url, name, assetType, moderationStrategy, projectName string) (map[string]interface{}, error) {
	active, err := s.getActiveKeyChecked()
	if err != nil {
		return nil, err
	}
	if groupID == "" {
		return nil, fmt.Errorf("groupId required")
	}
	if url == "" {
		return nil, fmt.Errorf("url required (publicly accessible image URL)")
	}
	if assetType == "" {
		assetType = "Image"
	}
	if projectName == "" {
		projectName = "default"
	}

	payload := map[string]interface{}{
		"GroupId":     groupID,
		"URL":         url,
		"AssetType":   assetType,
		"Name":        name,
		"ProjectName": projectName,
	}
	if moderationStrategy == "Skip" {
		payload["Moderation"] = map[string]string{"Strategy": "Skip"}
	}

	result, err := SignedFetch(SignedFetchInput{
		AK:      active.AK,
		SK:      active.SK,
		Region:  AssetsRegion,
		Service: AssetsService,
		Action:  "CreateAsset",
		Version: AssetsVersion,
		Body:    payload,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":        result["Id"],
		"groupId":   groupID,
		"name":      name,
		"status":    "Processing",
		"assetType": assetType,
	}, nil
}

func (s *Service) GetAsset(assetID, projectName string) (map[string]interface{}, error) {
	active, err := s.getActiveKeyChecked()
	if err != nil {
		return nil, err
	}
	if projectName == "" {
		projectName = "default"
	}
	return SignedFetch(SignedFetchInput{
		AK:      active.AK,
		SK:      active.SK,
		Region:  AssetsRegion,
		Service: AssetsService,
		Action:  "GetAsset",
		Version: AssetsVersion,
		Body: map[string]string{
			"Id":          assetID,
			"ProjectName": projectName,
		},
	})
}

func (s *Service) ListAssets(groupID, statuses, projectName string) (map[string]interface{}, error) {
	active, err := s.getActiveKeyChecked()
	if err != nil {
		return nil, err
	}
	if projectName == "" {
		projectName = "default"
	}
	filter := map[string]interface{}{
		"GroupType": "AIGC",
	}
	if groupID != "" {
		filter["GroupIds"] = []string{groupID}
	}
	if statuses != "" {
		filter["Statuses"] = strings.Split(statuses, ",")
	}

	result, err := SignedFetch(SignedFetchInput{
		AK:      active.AK,
		SK:      active.SK,
		Region:  AssetsRegion,
		Service: AssetsService,
		Action:  "ListAssets",
		Version: AssetsVersion,
		Body: map[string]interface{}{
			"Filter":     filter,
			"PageNumber": 1,
			"PageSize":   100,
			"SortBy":     "CreateTime",
			"SortOrder":  "Desc",
		},
	})
	if err != nil {
		return nil, err
	}
	items, _ := result["Items"].([]interface{})
	if items == nil {
		items = []interface{}{}
	}
	total, _ := result["TotalCount"].(float64)
	return map[string]interface{}{
		"assets": items,
		"total":  int(total),
	}, nil
}

func (s *Service) DeleteAsset(assetID, projectName string) (map[string]interface{}, error) {
	active, err := s.getActiveKeyChecked()
	if err != nil {
		return nil, err
	}
	if projectName == "" {
		projectName = "default"
	}
	_, err = SignedFetch(SignedFetchInput{
		AK:      active.AK,
		SK:      active.SK,
		Region:  AssetsRegion,
		Service: AssetsService,
		Action:  "DeleteAsset",
		Version: AssetsVersion,
		Body: map[string]string{
			"Id":          assetID,
			"ProjectName": projectName,
		},
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"ok":      true,
		"deleted": assetID,
	}, nil
}

// ─── Health & Debug ───────────────────────────────────────────

func (s *Service) Health() map[string]interface{} {
	data, _ := s.store.LoadKeys()
	var activeKey *APIKey
	if data.Active != "" {
		for _, k := range data.Keys {
			if k.ID == data.Active {
				activeKey = &k
				break
			}
		}
	}
	endpointID := DefaultEndpointID
	if activeKey != nil && activeKey.Endpoint != "" {
		endpointID = activeKey.Endpoint
	}
	ep := Endpoints[endpointID]
	return map[string]interface{}{
		"ok":                  true,
		"keysCount":           len(data.Keys),
		"activeKey":           activeKey != nil,
		"activeEndpoint":      endpointID,
		"activeEndpointLabel": ep.Label,
		"defaultModel":        DefaultModel,
	}
}

func (s *Service) Debug() map[string]interface{} {
	data, _ := s.store.LoadKeys()
	keys := make([]map[string]interface{}, len(data.Keys))
	for i, k := range data.Keys {
		ep := k.Endpoint
		if ep == "" {
			ep = "(missing)"
		}
		val := k.Value
		start := ""
		if len(val) > 4 {
			start = val[:4]
		}
		keys[i] = map[string]interface{}{
			"id":              k.ID,
			"name":            k.Name,
			"valuePreview":    maskKey(k.Value),
			"valueLength":     len(k.Value),
			"valueStartsWith": start,
			"endpoint":        ep,
			"isActive":        k.ID == data.Active,
		}
	}
	var activeEndpoint interface{}
	if data.Active != "" {
		for _, k := range data.Keys {
			if k.ID == data.Active {
				epID := k.Endpoint
				if epID == "" {
					epID = DefaultEndpointID
				}
				activeEndpoint = Endpoints[epID]
				break
			}
		}
	}
	return map[string]interface{}{
		"version": "v1.0",
		"keysFile": map[string]interface{}{
			"exists":         true,
			"activeId":       data.Active,
			"activeKeyFound": activeEndpoint != nil,
			"keysCount":      len(data.Keys),
			"keys":           keys,
		},
		"activeEndpoint": activeEndpoint,
		"endpoints":      Endpoints,
		"defaultModel":   DefaultModel,
	}
}
