package studio

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"dcs-back-v0/internal/character"
	"dcs-back-v0/internal/file"
	"dcs-back-v0/internal/provider"
	"dcs-back-v0/internal/studio/generators"
)

type Service struct {
	providerStore    *provider.Store
	fileService      *file.Service
	charService      *character.Service
	outputsDir       string
	handlers         []ModelHandler
	generatorsList   []generators.Generator
	tasks            map[string]*TaskRecord
	assetSyncStore   *AssetSyncStore
	baseURL          string
	logStore         *GenerationLogStore
	mu               sync.RWMutex
}

func NewService(providerStore *provider.Store, fileService *file.Service, outputsDir, baseURL string) *Service {
	return &Service{
		providerStore:  providerStore,
		fileService:    fileService,
		outputsDir:     outputsDir,
		baseURL:        baseURL,
		handlers:       []ModelHandler{},
		generatorsList: []generators.Generator{},
		tasks:          make(map[string]*TaskRecord),
	}
}

// ─── Legacy handler registration ──────────────────────────────────

func (s *Service) RegisterHandler(h ModelHandler) {
	s.handlers = append(s.handlers, h)
}

func (s *Service) pickHandler(modelName string) ModelHandler {
	for _, h := range s.handlers {
		if h.Matches(modelName) {
			return h
		}
	}
	return nil
}

// ─── Asset sync store ────────────────────────────────────────────

func (s *Service) SetAssetSyncStore(store *AssetSyncStore) {
	s.assetSyncStore = store
}

func (s *Service) SetCharacterService(cs *character.Service) {
	s.charService = cs
}

func (s *Service) SetLogStore(store *GenerationLogStore) {
	s.logStore = store
}

// ─── Generator registration ──────────────────────────────────────

func (s *Service) RegisterGenerator(gen generators.Generator) {
	s.generatorsList = append(s.generatorsList, gen)
}

func (s *Service) pickGenerator(modelName string) generators.Generator {
	for _, g := range s.generatorsList {
		if g.Match(modelName) {
			return g
		}
	}
	return nil
}

// ─── Unified payload generation ──────────────────────────────────

func (s *Service) GenerateUnified(req *StudioGenerateRequest) (*StudioGenerateResponse, error) {
	// Look up model by name (the user sends model name, not UUID)
	m, err := s.providerStore.GetModelByName(req.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("model not found: %s", req.Model)
	}

	// Resolve file IDs in content to data URLs (or asset:// URIs if synced)
	resolvedContent, err := s.resolveContent(req.Content, m.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve content: %w", err)
	}

	// Convert to generator request
	genReq := &generators.GeneratorRequest{
		Model:       m.Name,
		Content:     resolvedContent,
		Ratio:       req.Ratio,
		Duration:    int(req.Duration),
		CameraFixed: req.CameraFixed != nil && *req.CameraFixed,
		Seed:        req.Seed,
		Quality:     req.Quality,
		Quantity:    req.Quantity,
		Watermark:   req.Watermark != nil && *req.Watermark,
		Resolution:  req.Resolution,
		ImageMode:   req.ImageMode,
		APIKey:      m.APIKey,
		BaseURL:     m.URL,
		Endpoint:    m.Endpoint,
	}
	if req.GenerateAudio != nil {
		genReq.GenerateAudio = *req.GenerateAudio
	}

	// Pick generator
	gen := s.pickGenerator(m.Name)
	if gen == nil {
		return nil, fmt.Errorf("no generator available for model: %s", m.Name)
	}

	// Validate request against the generator
	if err := gen.Validate(genReq); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	result, err := gen.Generate(genReq)

	// ─── Save generation log (request + AI response) ──────────────
	if s.logStore != nil {
		reqBytes, _ := json.Marshal(req)
		genReqBytes, _ := json.Marshal(genReq)
		logEntry := &GenerationLog{
			TaskID:       "<error>",
			ModelName:    m.Name,
			Request:      string(reqBytes),
			Status:       "failed",
			ErrorMessage: "generation returned no result",
		}

		if result != nil {
			logEntry.TaskID = result.TaskID
			logEntry.Status = result.Status
			logEntry.ErrorMessage = ""
			logEntry.AICallPayload = string(genReqBytes)
			if result.Raw != nil {
				rawBytes, _ := json.Marshal(result.Raw)
				logEntry.AIResponse = string(rawBytes)
			}
			if len(result.Outputs) > 0 {
				outBytes, _ := json.Marshal(result.Outputs)
				logEntry.Outputs = string(outBytes)
			}
		}
		if err != nil {
			logEntry.Status = "failed"
			logEntry.ErrorMessage = err.Error()
		}

		if logEntry.TaskID != "<error>" {
			if saveErr := s.logStore.Create(logEntry); saveErr != nil {
				fmt.Printf("failed to save generation log: %v\n", saveErr)
			}
		}
	}
	// ──────────────────────────────────────────────────────────────

	if err != nil {
		return nil, err
	}

	// Track the task for status polling
	s.mu.Lock()
	s.tasks[result.TaskID] = &TaskRecord{
		TaskID:    result.TaskID,
		ModelID:   m.ID,
		ModelName: m.Name,
		Status:    result.Status,
		Result: &StatusResult{
			Status: result.Status,
			Raw:    result.Raw,
		},
	}
	s.mu.Unlock()

	// Convert outputs
	outputs := convertOutputs(result.Outputs)

	return &StudioGenerateResponse{
		TaskID:  result.TaskID,
		Model:   result.Model,
		Status:  result.Status,
		Outputs: outputs,
	}, nil
}

// ─── Legacy generation ───────────────────────────────────────────

func (s *Service) Generate(sel *Selection) (*GenerateResponse, error) {
	m, err := s.providerStore.GetModelByID(sel.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("model not found")
	}

	handler := s.pickHandler(m.Name)
	if handler == nil {
		return nil, fmt.Errorf("no handler available for model: %s", m.Name)
	}

	resp, err := handler.Generate(sel, m.APIKey, m.URL, m.Endpoint)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.tasks[resp.TaskID] = &TaskRecord{
		TaskID:    resp.TaskID,
		ModelID:   m.ID,
		ModelName: m.Name,
		Status:    "running",
	}
	s.mu.Unlock()

	return resp, nil
}

// ─── Asset sync ──────────────────────────────────────────────────

// SyncAsset uploads a local file to the model's asset library and stores the mapping.
func (s *Service) SyncAsset(req *SyncAssetRequest) (*SyncAssetResponse, error) {
	if s.assetSyncStore == nil {
		return nil, fmt.Errorf("asset sync store not available")
	}

	m, err := s.providerStore.GetModelByID(req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("model not found")
	}
	if m.AccessKeyID == "" || m.SecretAccessKey == "" {
		return nil, fmt.Errorf("model has no AK/SK configured. Set access_key_id and secret_access_key on the model")
	}
	if m.DefaultAssetGroupID == "" {
		return nil, fmt.Errorf("model has no default_asset_group_id. Create an asset group and set it on the model")
	}

	f, err := s.fileService.GetFile(req.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	if f == nil {
		return nil, fmt.Errorf("file not found")
	}

	// Create the sync record
	assetID := ""
	record := &ModelAsset{
		ModelID:      req.ModelID,
		FileID:       req.FileID,
		AssetGroupID: m.DefaultAssetGroupID,
		Status:       "syncing",
	}
	if err := s.assetSyncStore.Create(record); err != nil {
		return nil, fmt.Errorf("failed to create sync record: %w", err)
	}

	// Build the publicly accessible URL for the file
	fileURL := s.baseURL + "/api/v1/files/" + req.FileID + "/serve"

	// Upload to the asset library
	api := generators.NewAssetAPI(m.AccessKeyID, m.SecretAccessKey, m.DefaultAssetGroupID)
	result, err := api.CreateAsset(fileURL, f.Filename, detectAssetType(f.MimeType), "")
	if err != nil {
		s.assetSyncStore.UpdateStatus(record.ID, "failed", err.Error())
		return &SyncAssetResponse{
			ID:           record.ID,
			ModelID:      req.ModelID,
			FileID:       req.FileID,
			Status:       "failed",
			ErrorMessage: err.Error(),
		}, nil
	}

	assetID, _ = result["id"].(string)
	record.AssetID = assetID

	// Poll until Active (up to ~2 min)
	assetStatus := ""
	for i := 0; i < 20; i++ {
		statusResult, err := api.GetAsset(assetID, "")
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		assetStatus, _ = statusResult["Status"].(string)
		if assetStatus == "Active" || assetStatus == "Failed" {
			break
		}
		time.Sleep(3 * time.Second)
	}

	finalStatus := "active"
	errMsg := ""
	if assetStatus != "Active" {
		finalStatus = "failed"
		errMsg = fmt.Sprintf("asset did not become Active, last status: %s", assetStatus)
	}

	// Update the record
	if err := s.assetSyncStore.UpdateStatus(record.ID, finalStatus, errMsg); err != nil {
		return nil, fmt.Errorf("failed to update sync status: %w", err)
	}

	// Also update the in-memory record
	record.Status = finalStatus
	record.ErrorMessage = errMsg

	return &SyncAssetResponse{
		ID:           record.ID,
		ModelID:      req.ModelID,
		FileID:       req.FileID,
		AssetID:      assetID,
		AssetGroupID: m.DefaultAssetGroupID,
		Status:       finalStatus,
		ErrorMessage: errMsg,
	}, nil
}

// ListSyncedAssets returns all synced assets for a model.
func (s *Service) ListSyncedAssets(modelID string) ([]ModelAsset, error) {
	if s.assetSyncStore == nil {
		return nil, fmt.Errorf("asset sync store not available")
	}
	return s.assetSyncStore.ListByModel(modelID)
}

// GetSyncedAsset checks if a file is synced with a model.
func (s *Service) GetSyncedAsset(modelID, fileID string) (*ModelAsset, error) {
	if s.assetSyncStore == nil {
		return nil, nil
	}
	return s.assetSyncStore.GetByModelAndFile(modelID, fileID)
}

// ─── Enriched file listing ───────────────────────────────────────

// resolveModelBriefs resolves a set of model IDs to ModelBrief objects.
func (s *Service) resolveModelBriefs(modelIDs map[string]bool) []ModelBrief {
	var briefs []ModelBrief
	for id := range modelIDs {
		m, err := s.providerStore.GetModelByID(id)
		if err != nil || m == nil {
			briefs = append(briefs, ModelBrief{ModelID: id, Name: "unknown"})
			continue
		}
		briefs = append(briefs, ModelBrief{ModelID: id, Name: m.Name})
	}
	return briefs
}

// GetFilesWithSync returns files with their synced model info.
func (s *Service) GetFilesWithSync(category, storage string, trashed bool) ([]FileWithSync, error) {
	files, err := s.fileService.ListFiles(category, storage, trashed)
	if err != nil {
		return nil, err
	}

	if s.assetSyncStore == nil {
		// Return files without sync info
		result := make([]FileWithSync, len(files))
		for i, f := range files {
			result[i] = fileToFileWithSync(f, nil)
		}
		return result, nil
	}

	fileIDs := make([]string, len(files))
	for i, f := range files {
		fileIDs[i] = f.ID
	}

	syncMap, err := s.assetSyncStore.GetByFileIDs(fileIDs)
	if err != nil {
		return nil, err
	}

	result := make([]FileWithSync, len(files))
	for i, f := range files {
		briefs := s.modelAssetsToBriefs(syncMap[f.ID])
		result[i] = fileToFileWithSync(f, briefs)
	}
	return result, nil
}

// GetCharacterFilesWithSync returns a character's files with their synced model info.
func (s *Service) GetCharacterFilesWithSync(characterID string) ([]CharacterFileWithSync, error) {
	if s.charService == nil {
		return nil, fmt.Errorf("character service not available")
	}

	files, err := s.charService.ListFiles(characterID)
	if err != nil {
		return nil, err
	}

	if s.assetSyncStore == nil {
		result := make([]CharacterFileWithSync, len(files))
		for i, f := range files {
			result[i] = charFileToCharFileWithSync(f, nil)
		}
		return result, nil
	}

	fileIDs := make([]string, len(files))
	for i, f := range files {
		fileIDs[i] = f.FileID
	}

	syncMap, err := s.assetSyncStore.GetByFileIDs(fileIDs)
	if err != nil {
		return nil, err
	}

	result := make([]CharacterFileWithSync, len(files))
	for i, f := range files {
		briefs := s.modelAssetsToBriefs(syncMap[f.FileID])
		result[i] = charFileToCharFileWithSync(f, briefs)
	}
	return result, nil
}

// SyncCharacterAssets syncs all files linked to a character to a model's asset library.
func (s *Service) SyncCharacterAssets(req *SyncCharacterRequest) (*SyncResultSummary, error) {
	if s.charService == nil {
		return nil, fmt.Errorf("character service not available")
	}

	// Verify model exists and has AK/SK
	m, err := s.providerStore.GetModelByID(req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("model not found")
	}
	if m.AccessKeyID == "" || m.SecretAccessKey == "" {
		return nil, fmt.Errorf("model has no AK/SK configured")
	}
	if m.DefaultAssetGroupID == "" {
		return nil, fmt.Errorf("model has no default_asset_group_id")
	}

	// Get character files
	charFiles, err := s.charService.ListFiles(req.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character files: %w", err)
	}

	var results []SyncAssetResponse
	for _, cf := range charFiles {
		r, err := s.SyncAsset(&SyncAssetRequest{
			ModelID: req.ModelID,
			FileID:  cf.FileID,
		})
		if err != nil {
			results = append(results, SyncAssetResponse{
				FileID:       cf.FileID,
				Status:       "failed",
				ErrorMessage: err.Error(),
			})
			continue
		}
		results = append(results, *r)
	}

	summary := &SyncResultSummary{
		ModelID:    req.ModelID,
		Total:      len(charFiles),
		Successful: 0,
		Failed:     0,
		Results:    results,
	}
	for _, r := range results {
		if r.Status == "active" {
			summary.Successful++
		} else {
			summary.Failed++
		}
	}
	return summary, nil
}

// ─── Helpers ─────────────────────────────────────────────────────

func (s *Service) modelAssetsToBriefs(assets []ModelAsset) []ModelBrief {
	if len(assets) == 0 {
		return nil
	}
	modelIDs := make(map[string]bool)
	for _, a := range assets {
		modelIDs[a.ModelID] = true
	}
	return s.resolveModelBriefs(modelIDs)
}

func fileToFileWithSync(f file.File, briefs []ModelBrief) FileWithSync {
	return FileWithSync{
		ID:           f.ID,
		Filename:     f.Filename,
		Path:         f.Path,
		Size:         f.Size,
		MimeType:     f.MimeType,
		Category:     f.Category,
		Format:       f.Format,
		Storage:      f.Storage,
		Trashed:      f.Trashed,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
		DeletedAt:    f.DeletedAt,
		SyncedModels: briefs,
	}
}

func charFileToCharFileWithSync(f character.CharacterFile, briefs []ModelBrief) CharacterFileWithSync {
	return CharacterFileWithSync{
		FileID:       f.FileID,
		Role:         f.Role,
		Filename:     f.Filename,
		URL:          f.URL,
		ThumbnailURL: f.ThumbnailURL,
		MimeType:     f.MimeType,
		Category:     f.Category,
		Format:       f.Format,
		Size:         f.Size,
		CreatedAt:    f.CreatedAt,
		SyncedModels: briefs,
	}
}

// ─── Status and cancellation (shared) ────────────────────────────

func (s *Service) GetStatus(taskID string) (*StatusResult, error) {
	s.mu.RLock()
	record, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown task: %s", taskID)
	}

	m, err := s.providerStore.GetModelByID(record.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("model for task %s not found", taskID)
	}

	// Try generator first, then fall back to legacy handler
	gen := s.pickGenerator(m.Name)
	if gen != nil {
		result, err := gen.GetStatus(taskID, m.APIKey, m.URL, m.Endpoint)
		if err != nil {
			return nil, err
		}
		statusResult := &StatusResult{
			Status: result.Status,
			Error:  result.Error,
			Raw:    result.Raw,
		}
		if len(result.Outputs) > 0 {
			statusResult.VideoURL = result.Outputs[0].URL
			statusResult.LocalURL = result.Outputs[0].LocalURL
		}
		if result.Status == "succeeded" || result.Status == "failed" {
			s.mu.Lock()
			record.Status = result.Status
			record.Result = statusResult
			s.mu.Unlock()

			// Update generation log with final AI response
			s.updateLogWithFinalStatus(taskID, result)
		}
		return statusResult, nil
	}

	// Fall back to legacy handler
	handler := s.pickHandler(m.Name)
	if handler == nil {
		return nil, fmt.Errorf("no handler available for model: %s", m.Name)
	}

	result, err := handler.GetStatus(taskID, m.APIKey, m.URL, m.Endpoint)
	if err != nil {
		return nil, err
	}

	if result.Status == "succeeded" || result.Status == "failed" {
		s.mu.Lock()
		record.Status = result.Status
		record.Result = result
		s.mu.Unlock()
	}

	return result, nil
}

func (s *Service) GetStatusUnified(taskID string) (*StudioStatusResponse, error) {
	sr, err := s.GetStatus(taskID)
	if err != nil {
		return nil, err
	}

	resp := &StudioStatusResponse{
		Status: sr.Status,
		Error:  sr.Error,
		Outputs: []OutputResource{},
	}

	if sr.VideoURL != "" {
		resp.Outputs = append(resp.Outputs, OutputResource{
			URL:      sr.VideoURL,
			LocalURL: sr.LocalURL,
			Type:     "video",
		})
	}
	if sr.ImageURL != "" {
		resp.Outputs = append(resp.Outputs, OutputResource{
			URL:  sr.ImageURL,
			Type: "image",
		})
	}

	return resp, nil
}

func (s *Service) CancelTask(taskID string) error {
	s.mu.RLock()
	record, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown task: %s", taskID)
	}

	m, err := s.providerStore.GetModelByID(record.ModelID)
	if err != nil {
		return err
	}
	if m == nil {
		return fmt.Errorf("model for task %s not found", taskID)
	}

	// Try generator first
	gen := s.pickGenerator(m.Name)
	if gen != nil {
		return gen.CancelTask(taskID, m.APIKey, m.URL, m.Endpoint)
	}

	// Fall back to legacy handler
	handler := s.pickHandler(m.Name)
	if handler == nil {
		return fmt.Errorf("no handler available for model: %s", m.Name)
	}

	return handler.CancelTask(taskID, m.APIKey, m.URL, m.Endpoint)
}

// ─── Log listing ─────────────────────────────────────────────────

// ListGenerationLogs returns paginated generation logs.
func (s *Service) ListGenerationLogs(page, limit int) (*ListGenerationLogsResponse, error) {
	if s.logStore == nil {
		return nil, fmt.Errorf("log store not available")
	}

	logs, total, err := s.logStore.List(page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list generation logs: %w", err)
	}

	totalPages := (total + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	return &ListGenerationLogsResponse{
		Logs:       logs,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// GetGenerationLog returns a single generation log by ID.
func (s *Service) GetGenerationLog(id string) (*GenerationLog, error) {
	if s.logStore == nil {
		return nil, fmt.Errorf("log store not available")
	}

	log, err := s.logStore.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get generation log: %w", err)
	}
	if log == nil {
		return nil, fmt.Errorf("generation log not found: %s", id)
	}

	return log, nil
}

// ─── Helpers ─────────────────────────────────────────────────────

// updateLogWithFinalStatus updates the generation log with the final AI response
// when an async task completes (succeeded or failed).
func (s *Service) updateLogWithFinalStatus(taskID string, result *generators.GeneratorResult) {
	if s.logStore == nil {
		return
	}

	log, logErr := s.logStore.GetByTaskID(taskID)
	if logErr != nil || log == nil {
		// No log entry (e.g. legacy path) — skip
		return
	}

	aiResponse := ""
	if result.Raw != nil {
		rawBytes, _ := json.Marshal(result.Raw)
		aiResponse = string(rawBytes)
	}

	outputs := ""
	if len(result.Outputs) > 0 {
		outBytes, _ := json.Marshal(result.Outputs)
		outputs = string(outBytes)
	}

	errorMsg := result.Error

	if saveErr := s.logStore.UpdateByTaskID(taskID, aiResponse, outputs, result.Status, errorMsg); saveErr != nil {
		fmt.Printf("failed to update generation log for task %s: %v\n", taskID, saveErr)
	}
}

func (s *Service) resolveContent(items []ContentItem, modelID string) ([]generators.ContentItem, error) {
	resolved := make([]generators.ContentItem, len(items))
	for i, item := range items {
		ci := generators.ContentItem{
			Type: item.Type,
			Text: item.Text,
			Name: item.Name,
			ID:   item.ID,
		}

		if item.Type != "text" && item.ID != "" {
			// Check if file is synced to this model's asset library
			if modelID != "" && s.assetSyncStore != nil {
				synced, err := s.assetSyncStore.GetByModelAndFile(modelID, item.ID)
				if err == nil && synced != nil && synced.Status == "active" && synced.AssetID != "" {
					ci.DataURL = "asset://" + synced.AssetID
					resolved[i] = ci
					continue
				}
			}

			// Not synced — fall back to data URL
			f, err := s.fileService.GetFile(item.ID)
			if err != nil {
				return nil, fmt.Errorf("content[%d] file %s: %w", i, item.ID, err)
			}
			if f == nil {
				return nil, fmt.Errorf("content[%d] file %s not found", i, item.ID)
			}

			path, err := s.fileService.GetServePath(item.ID)
			if err != nil {
				return nil, fmt.Errorf("content[%d] failed to resolve path for %s: %w", i, item.ID, err)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("content[%d] failed to read file %s: %w", i, item.ID, err)
			}

			mimeType := f.MimeType
			if mimeType == "" {
				switch strings.ToLower(f.Format) {
				case "png":
					mimeType = "image/png"
				case "jpg", "jpeg":
					mimeType = "image/jpeg"
				case "gif":
					mimeType = "image/gif"
				case "webp":
					mimeType = "image/webp"
				case "mp4":
					mimeType = "video/mp4"
				case "mp3":
					mimeType = "audio/mpeg"
				default:
					mimeType = "application/octet-stream"
				}
			}

			ci.DataURL = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
		}
		resolved[i] = ci
	}
	return resolved, nil
}

func detectAssetType(mimeType string) string {
	if strings.HasPrefix(mimeType, "video") {
		return "Video"
	}
	if strings.HasPrefix(mimeType, "audio") {
		return "Audio"
	}
	return "Image"
}

func convertOutputs(src []generators.OutputResource) []OutputResource {
	if src == nil {
		return []OutputResource{}
	}
	dst := make([]OutputResource, len(src))
	for i, o := range src {
		dst[i] = OutputResource{
			URL:      o.URL,
			LocalURL: o.LocalURL,
			Type:     o.Type,
		}
	}
	return dst
}
