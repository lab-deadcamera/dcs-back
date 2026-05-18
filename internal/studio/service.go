package studio

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"dcs-back-v0/internal/character"
	"dcs-back-v0/internal/file"
	"dcs-back-v0/internal/provider"
)

// PipelineRunner is the internal interface satisfied by all domain generators
// (video.VideoGenerator, image.ImageGenerator, etc.) for the unified pipeline.
type PipelineRunner interface {
	Match(modelName string) bool
	Validate(req *GeneratorRequest) error
	Generate(req *GeneratorRequest) (*GeneratorResult, error)
	GetStatus(taskID, apiKey, baseURL, endpoint string) (*GeneratorResult, error)
	CancelTask(taskID, apiKey, baseURL, endpoint string) error
	BuildPayload(req *GeneratorRequest) map[string]interface{}
	ContentType() string
	Name() string
}

type Service struct {
	providerStore      *provider.Store
	fileService        *file.Service
	charService        *character.Service
	outputsDir         string
	handlers           []ModelHandler
	pipelineGens       []PipelineRunner
	tasks              map[string]*TaskRecord
	assetSyncStore     *AssetSyncStore
	baseURL            string
	logStore           *GenerationLogStore
	commStore          *ServerCommunicationStore
	assetStore         *GeneratedAssetStore
	assetAccessKeyID     string
	assetSecretAccessKey string
	assetDefaultGroupID  string
	mu                 sync.RWMutex
}

func NewService(providerStore *provider.Store, fileService *file.Service, outputsDir, baseURL string) *Service {
	return &Service{
		providerStore: providerStore,
		fileService:   fileService,
		outputsDir:    outputsDir,
		baseURL:       baseURL,
		handlers:      []ModelHandler{},
		pipelineGens:  []PipelineRunner{},
		tasks:         make(map[string]*TaskRecord),
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

func (s *Service) SetGeneratedAssetStore(store *GeneratedAssetStore) {
	s.assetStore = store
}

func (s *Service) SetAssetCredentials(accessKeyID, secretAccessKey, defaultGroupID string) {
	s.assetAccessKeyID = accessKeyID
	s.assetSecretAccessKey = secretAccessKey
	s.assetDefaultGroupID = defaultGroupID
}

// effectiveCredentials returns the AK/SK/groupID to use for asset operations.
// Prefers per-model values from the DB, falls back to globally configured env vars.
func (s *Service) effectiveCredentials(m *provider.Model) (accessKeyID, secretAccessKey, defaultGroupID string) {
	accessKeyID = m.AccessKeyID
	if accessKeyID == "" {
		accessKeyID = s.assetAccessKeyID
	}
	secretAccessKey = m.SecretAccessKey
	if secretAccessKey == "" {
		secretAccessKey = s.assetSecretAccessKey
	}
	defaultGroupID = m.DefaultAssetGroupID
	if defaultGroupID == "" {
		defaultGroupID = s.assetDefaultGroupID
	}
	log.Printf("[gallery-sync] effectiveCredentials model=%q db_ak=%q env_ak=%q final_ak=%q", m.Name, m.AccessKeyID, s.assetAccessKeyID, accessKeyID)
	return
}

// ─── Generator registration ──────────────────────────────────────

// RegisterGenerator registers a generator that satisfies the PipelineRunner interface.
// Both video.VideoGenerator and image.ImageGenerator match structurally.
func (s *Service) RegisterGenerator(gen PipelineRunner) {
	s.pipelineGens = append(s.pipelineGens, gen)
}

func (s *Service) pickGenerator(modelName string) PipelineRunner {
	for _, g := range s.pipelineGens {
		if g.Match(modelName) {
			return g
		}
	}
	return nil
}

// ─── Unified payload generation ──────────────────────────────────

func (s *Service) GenerateUnified(req *StudioGenerateRequest) (*StudioGenerateResponse, error) {
	// Validar que los campos de sesión estén presentes (obligatorios para tracking).
	if req.ProjectID == "" || req.SceneID == "" || req.SceneCode == "" || req.TakeNumber <= 0 {
		return nil, fmt.Errorf("project_id, scene_id, scene_code and take_number are required for generation")
	}

	var (
		genReq    *GeneratorRequest
		modelName string
		taskID    string
		status    = "failed"
		aiResp    string
		outputs   string
		aiCall    string
		errLog    string
	)

	// Defer log save — runs on every return path (including early errors)
	defer func() {
		if s.logStore == nil {
			return
		}
		reqBytes, _ := json.Marshal(req)
		if modelName == "" {
			modelName = req.Model
		}
		if modelName == "" {
			return
		}
		if taskID == "" {
			taskID = "<no-task>"
		}
		logEntry := &GenerationLog{
			TaskID:        taskID,
			ModelName:     modelName,
			UserID:        intPtrOrNil(req.UserID),
			ProjectID:     req.ProjectID,
			SceneID:       req.SceneID,
			SceneCode:     req.SceneCode,
			TakeNumber:    req.TakeNumber,
			Request:       string(reqBytes),
			AIResponse:    aiResp,
			AICallPayload: aiCall,
			Outputs:       outputs,
			Status:        status,
			ErrorMessage:  errLog,
		}
		if saveErr := s.logStore.Create(logEntry); saveErr != nil {
			fmt.Printf("failed to save generation log: %v\n", saveErr)
		}
	}()

	// Look up model by name
	m, err := s.providerStore.GetModelByName(req.Model)
	if err != nil {
		errLog = fmt.Sprintf("failed to get model: %v", err)
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		errLog = fmt.Sprintf("model not found: %s", req.Model)
		return nil, fmt.Errorf("model not found: %s", req.Model)
	}
	modelName = m.Name

	// Resolve file IDs in content to data URLs (or asset:// URIs if synced)
	resolvedContent, err := s.resolveContent(req.Content, m.ID)
	if err != nil {
		errLog = fmt.Sprintf("failed to resolve content: %v", err)
		return nil, fmt.Errorf("failed to resolve content: %w", err)
	}

	// Convert to generator request
	genReq = &GeneratorRequest{
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
		errLog = fmt.Sprintf("no generator available for model: %s", m.Name)
		return nil, fmt.Errorf("no generator available for model: %s", m.Name)
	}

	// Validate request against the generator
	if err := gen.Validate(genReq); err != nil {
		errLog = fmt.Sprintf("invalid request: %v", err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	result, err := gen.Generate(genReq)

	// Capture result data for the log
	genReqBytes, _ := json.Marshal(genReq)
	aiCall = string(genReqBytes)
	if result != nil {
		taskID = result.TaskID
		status = result.Status
		if result.Raw != nil {
			rawBytes, _ := json.Marshal(result.Raw)
			aiResp = string(rawBytes)
		}
		if len(result.Outputs) > 0 {
			outBytes, _ := json.Marshal(result.Outputs)
			outputs = string(outBytes)
		}
	}
	if err != nil {
		errLog = err.Error()
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

	out := convertOutputs(result.Outputs)

	return &StudioGenerateResponse{
		TaskID:  result.TaskID,
		Model:   result.Model,
		Status:  result.Status,
		Outputs: out,
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
	log.Printf("[sync-asset] SyncAsset start model_id=%q file_id=%q", req.ModelID, req.FileID)

	if s.assetSyncStore == nil {
		log.Printf("[sync-asset] assetSyncStore not available")
		return nil, fmt.Errorf("asset sync store not available")
	}

	m, err := s.providerStore.GetModelByID(req.ModelID)
	if err != nil {
		log.Printf("[sync-asset] GetModelByID error: %v", err)
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		log.Printf("[sync-asset] model %q not found", req.ModelID)
		return nil, fmt.Errorf("model not found")
	}

	ak, sk, groupID := s.effectiveCredentials(m)
	log.Printf("[sync-asset] model=%q group_id=%q ak_set=%v sk_set=%v", m.Name, groupID, ak != "", sk != "")
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("model has no AK/SK configured. Set access_key_id and secret_access_key on the model or ASSET_ACCESS_KEY_ID / ASSET_SECRET_ACCESS_KEY env vars")
	}
	if groupID == "" {
		return nil, fmt.Errorf("no asset group configured. Set default_asset_group_id on the model or ASSET_DEFAULT_GROUP_ID env var")
	}

	f, err := s.fileService.GetFile(req.FileID)
	if err != nil {
		log.Printf("[sync-asset] GetFile error: %v", err)
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	if f == nil {
		log.Printf("[sync-asset] file %q not found", req.FileID)
		return nil, fmt.Errorf("file not found")
	}
	log.Printf("[sync-asset] file found name=%q mime=%q", f.Filename, f.MimeType)

	// Create the sync record
	assetID := ""
	record := &ModelAsset{
		ModelID:      req.ModelID,
		FileID:       req.FileID,
		AssetGroupID: groupID,
		Status:       "syncing",
	}
	if err := s.assetSyncStore.Create(record); err != nil {
		log.Printf("[sync-asset] Create sync record error: %v", err)
		return nil, fmt.Errorf("failed to create sync record: %w", err)
	}
	log.Printf("[sync-asset] sync record created id=%s", record.ID)

	// Build the publicly accessible URL for the file
	fileURL := s.baseURL + "/api/v1/files/" + req.FileID + "/serve"

	// Upload to the asset library
	log.Printf("[sync-asset] calling CreateAsset url=%q filename=%q type=%q", fileURL, f.Filename, detectAssetType(f.MimeType))
	api := NewAssetAPI(ak, sk, groupID)
	result, err := api.CreateAsset(fileURL, f.Filename, detectAssetType(f.MimeType), "")
	if err != nil {
		log.Printf("[sync-asset] CreateAsset FAILED: %v", err)
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
	log.Printf("[sync-asset] CreateAsset OK asset_id=%s", assetID)

	// Poll until Active (up to ~2 min)
	log.Printf("[sync-asset] polling asset %s for Active status", assetID)
	assetStatus := ""
	for i := 0; i < 20; i++ {
		statusResult, err := api.GetAsset(assetID, "")
		if err != nil {
			log.Printf("[sync-asset] poll[%d] GetAsset error: %v", i, err)
			time.Sleep(3 * time.Second)
			continue
		}
		assetStatus, _ = statusResult["Status"].(string)
		log.Printf("[sync-asset] poll[%d] status=%q", i, assetStatus)
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
		log.Printf("[sync-asset] final status NOT Active: %q", assetStatus)
	} else {
		log.Printf("[sync-asset] asset is now Active")
	}

	// Update the record
	if err := s.assetSyncStore.UpdateStatus(record.ID, finalStatus, errMsg); err != nil {
		return nil, fmt.Errorf("failed to update sync status: %w", err)
	}

	// Also update the in-memory record
	record.Status = finalStatus
	record.ErrorMessage = errMsg

	log.Printf("[sync-asset] SyncAsset done record_id=%s asset_id=%s status=%s", record.ID, assetID, finalStatus)
	return &SyncAssetResponse{
		ID:           record.ID,
		ModelID:      req.ModelID,
		FileID:       req.FileID,
		AssetID:      assetID,
		AssetGroupID: groupID,
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
// Validates existing groups and assets before creating new ones.
func (s *Service) SyncCharacterAssets(req *SyncCharacterRequest) (*SyncResultSummary, error) {
	log.Printf("[sync-char] SyncCharacterAssets start model_id=%q character_id=%q", req.ModelID, req.CharacterID)

	if s.charService == nil {
		log.Printf("[sync-char] charService not available")
		return nil, fmt.Errorf("character service not available")
	}

	// Verify model exists and has AK/SK
	m, err := s.providerStore.GetModelByID(req.ModelID)
	if err != nil {
		log.Printf("[sync-char] GetModelByID error: %v", err)
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		log.Printf("[sync-char] model %q not found", req.ModelID)
		return nil, fmt.Errorf("model not found")
	}

	ak, sk, _ := s.effectiveCredentials(m)
	log.Printf("[sync-char] model=%q ak_set=%v sk_set=%v", m.Name, ak != "", sk != "")
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("model has no AK/SK configured. Set access_key_id and secret_access_key on the model or ASSET_ACCESS_KEY_ID / ASSET_SECRET_ACCESS_KEY env vars")
	}

	// Get character info for the asset group name and description
	char, err := s.charService.GetByID(req.CharacterID)
	if err != nil {
		log.Printf("[sync-char] GetByID error: %v", err)
		return nil, fmt.Errorf("failed to get character: %w", err)
	}
	if char == nil {
		log.Printf("[sync-char] character %q not found", req.CharacterID)
		return nil, fmt.Errorf("character not found")
	}
	log.Printf("[sync-char] character found name=%q", char.Name)

	// Get character files
	charFiles, err := s.charService.ListFiles(req.CharacterID)
	if err != nil {
		log.Printf("[sync-char] ListFiles error: %v", err)
		return nil, fmt.Errorf("failed to get character files: %w", err)
	}
	if len(charFiles) == 0 {
		log.Printf("[sync-char] no files for character %q", req.CharacterID)
		return &SyncResultSummary{
			ModelID:    req.ModelID,
			Total:      0,
			Successful: 0,
			Failed:     0,
			Results:    []SyncAssetResponse{},
		}, nil
	}
	log.Printf("[sync-char] character has %d files", len(charFiles))

	// Collect file IDs for batch lookup
	fileIDs := make([]string, len(charFiles))
	for i, cf := range charFiles {
		fileIDs[i] = cf.FileID
	}

	// Check existing sync records to find an existing group or determine what needs syncing
	syncMap, _ := s.assetSyncStore.GetByFileIDs(fileIDs)

	// Look for an existing asset group ID from previous syncs
	var groupID string
	for _, assets := range syncMap {
		for _, a := range assets {
			if a.ModelID == req.ModelID && a.AssetGroupID != "" {
				groupID = a.AssetGroupID
				break
			}
		}
		if groupID != "" {
			break
		}
	}
	log.Printf("[sync-char] existing group_id=%q", groupID)

	// Create API client (with or without existing group)
	api := NewAssetAPI(ak, sk, groupID)

	// Create asset group only if none exists for this character+model
	if groupID == "" {
		log.Printf("[sync-char] no existing group, creating asset group for character %q", char.Name)
		groupResult, err := api.CreateAssetGroup(char.Name, char.Description, "")
		if err != nil {
			log.Printf("[sync-char] CreateAssetGroup error: %v", err)
			return nil, fmt.Errorf("failed to create asset group: %w", err)
		}
		groupID, _ = groupResult["id"].(string)
		if groupID == "" {
			return nil, fmt.Errorf("no asset group ID returned from CreateAssetGroup")
		}
		log.Printf("[sync-char] created asset group id=%s", groupID)
		api = NewAssetAPI(ak, sk, groupID)
	}

	// Process each file — skip if already synced, upload if new or failed
	var results []SyncAssetResponse
	for _, cf := range charFiles {
		existing := syncMap[cf.FileID]

		// Check if this file is already synced and active for this model
		alreadySynced := false
		for _, a := range existing {
			if a.ModelID == req.ModelID && a.Status == "active" && a.AssetID != "" {
				log.Printf("[sync-char] file %q already synced asset_id=%s", cf.FileID, a.AssetID)
				alreadySynced = true
				results = append(results, SyncAssetResponse{
					ID:           a.ID,
					ModelID:      req.ModelID,
					FileID:       cf.FileID,
					AssetID:      a.AssetID,
					AssetGroupID: groupID,
					Status:       "active",
				})
				break
			}
		}
		if alreadySynced {
			continue
		}

		// Not synced or previously failed — upload
		log.Printf("[sync-char] uploading file %q to group %s", cf.FileID, groupID)
		r, err := s.uploadAndTrackAsset(req.ModelID, cf.FileID, groupID, api)
		if err != nil {
			log.Printf("[sync-char] uploadAndTrackAsset FAILED file=%q err=%v", cf.FileID, err)
			results = append(results, SyncAssetResponse{
				FileID:       cf.FileID,
				Status:       "failed",
				ErrorMessage: err.Error(),
			})
			continue
		}
		log.Printf("[sync-char] uploadAndTrackAsset OK file=%q status=%s asset_id=%s", cf.FileID, r.Status, r.AssetID)
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
	log.Printf("[sync-char] SyncCharacterAssets done total=%d ok=%d failed=%d", summary.Total, summary.Successful, summary.Failed)
	return summary, nil
}

// uploadAndTrackAsset uploads a file to the BytePlus asset library via CreateAsset,
// polls until Active or Failed, and stores the mapping in model_assets.
func (s *Service) uploadAndTrackAsset(modelID, fileID, groupID string, api *AssetAPI) (*SyncAssetResponse, error) {
	if s.assetSyncStore == nil {
		return nil, fmt.Errorf("asset sync store not available")
	}

	f, err := s.fileService.GetFile(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	if f == nil {
		return nil, fmt.Errorf("file not found")
	}

	// Create the sync record (asset_id empty until upload completes)
	record := &ModelAsset{
		ModelID:      modelID,
		FileID:       fileID,
		AssetGroupID: groupID,
		Status:       "syncing",
	}
	if err := s.assetSyncStore.Create(record); err != nil {
		return nil, fmt.Errorf("failed to create sync record: %w", err)
	}

	// Build the publicly accessible file URL
	fileURL := s.baseURL + "/api/v1/files/" + fileID + "/serve"

	// Upload to the asset library
	result, err := api.CreateAsset(fileURL, f.Filename, detectAssetType(f.MimeType), "")
	if err != nil {
		s.assetSyncStore.UpdateStatus(record.ID, "failed", err.Error())
		return &SyncAssetResponse{
			ID:           record.ID,
			ModelID:      modelID,
			FileID:       fileID,
			Status:       "failed",
			ErrorMessage: err.Error(),
		}, nil
	}

	assetID, _ := result["id"].(string)
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

	record.Status = finalStatus
	record.ErrorMessage = errMsg

	return &SyncAssetResponse{
		ID:           record.ID,
		ModelID:      modelID,
		FileID:       fileID,
		AssetID:      assetID,
		AssetGroupID: groupID,
		Status:       finalStatus,
		ErrorMessage: errMsg,
	}, nil
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

// statusFromLog recupera el estado de una tarea desde el log cuando
// la tarea ya no está en memoria (ej. reinicio del servidor).
func (s *Service) statusFromLog(log *GenerationLog) (*StatusResult, error) {
	// Buscar el modelo por nombre
	m, err := s.providerStore.GetModelByName(log.ModelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get model for task %s: %w", log.TaskID, err)
	}
	if m == nil {
		// No hay modelo configurado, devolver el estado del log
		return &StatusResult{Status: log.Status, Error: "model not found for task " + log.TaskID}, nil
	}

	gen := s.pickGenerator(m.Name)
	if gen == nil {
		// Sin generator disponible, devolver el estado del log
		return &StatusResult{Status: log.Status}, nil
	}

	result, err := gen.GetStatus(log.TaskID, m.APIKey, m.URL, m.Endpoint)
	if err != nil {
		// Error consultando al generator, devolver el estado del log
		return &StatusResult{Status: log.Status, Error: err.Error()}, nil
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
		s.updateLogWithFinalStatus(log.TaskID, result)
		if result.Status == "succeeded" {
			s.saveGeneratedAssets(log.TaskID, result)
		}
	}

	return statusResult, nil
}

func (s *Service) GetStatus(taskID string) (*StatusResult, error) {
	s.mu.RLock()
	record, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if !ok {
		// Task not in memory (e.g. after restart) — try to recover from log
		log, logErr := s.logStore.GetByTaskID(taskID)
		if logErr != nil || log == nil {
			return nil, fmt.Errorf("unknown task: %s", taskID)
		}
		resp, err := s.statusFromLog(log)
		if err != nil {
			return nil, err
		}
		return resp, nil
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
if result.Status == "succeeded" {
					s.saveGeneratedAssets(taskID, result)
				}
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
		Status:  sr.Status,
		Error:   sr.Error,
		Raw:     sr.Raw,
		Outputs: []OutputResource{},
	}

	// Extract optional progress from raw response
	if rawMap, ok := sr.Raw.(map[string]interface{}); ok {
		for _, key := range []string{"progress", "percentage", "task_progress"} {
			if v, exists := rawMap[key]; exists {
				resp.Progress = v
				break
			}
		}
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

// ListGenerationLogs returns paginated generation logs, optionally filtered.
func (s *Service) ListGenerationLogs(page, limit int, projectID, sceneID, status, modelName string, userID int, dateFrom, dateTo string) (*ListGenerationLogsResponse, error) {
	if s.logStore == nil {
		return nil, fmt.Errorf("log store not available")
	}

	var (
		logs  []GenerationLog
		total int
		err   error
	)
	if projectID != "" || sceneID != "" || status != "" || modelName != "" || userID > 0 || dateFrom != "" || dateTo != "" {
		logs, total, err = s.logStore.ListByFilter(page, limit, projectID, sceneID, status, modelName, userID, dateFrom, dateTo)
	} else {
		logs, total, err = s.logStore.List(page, limit)
	}
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

// intPtrOrNil returns a pointer to v if v > 0, otherwise nil.
func intPtrOrNil(v int) *int {
	if v <= 0 {
		return nil
	}
	return &v
}

// ─── Preview (dry-run) ───────────────────────────────────────────

// PreviewPayload builds the AI API payload without sending it or saving logs.
func (s *Service) PreviewPayload(req *StudioGenerateRequest) (*PreviewPayloadResponse, error) {
	m, err := s.providerStore.GetModelByName(req.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("model not found: %s", req.Model)
	}

	resolvedContent, err := s.resolveContent(req.Content, m.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve content: %w", err)
	}

	genReq := &GeneratorRequest{
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
	}
	if req.GenerateAudio != nil {
		genReq.GenerateAudio = *req.GenerateAudio
	}

	gen := s.pickGenerator(m.Name)
	if gen == nil {
		return nil, fmt.Errorf("no generator available for model: %s", m.Name)
	}

	payload := gen.BuildPayload(genReq)
	return &PreviewPayloadResponse{
		Model:       m.Name,
		Endpoint:    m.Endpoint,
		Payload:     payload,
		ContentType: gen.ContentType(),
	}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────

// updateLogWithFinalStatus updates the generation log with the final AI response
// when an async task completes (succeeded or failed).
func (s *Service) updateLogWithFinalStatus(taskID string, result *GeneratorResult) {
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

func (s *Service) resolveContent(items []ContentItem, modelID string) ([]ContentItem, error) {
	resolved := make([]ContentItem, len(items))
	for i, item := range items {
		ci := ContentItem{
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

			// Verify file exists
			f, err := s.fileService.GetFile(item.ID)
			if err != nil {
				return nil, fmt.Errorf("content[%d] file %s: %w", i, item.ID, err)
			}
			if f == nil {
				return nil, fmt.Errorf("content[%d] file %s not found", i, item.ID)
			}

			// Use public serve URL instead of base64 data URL
			ci.DataURL = s.baseURL + "/api/v1/files/" + item.ID + "/serve"
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

func convertOutputs(src []OutputResource) []OutputResource {
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

func (s *Service) SetCommStore(store *ServerCommunicationStore) {
	s.commStore = store
}

func (s *Service) ListServerCommunications(taskID, modelName string, page, limit int) (*ServerCommListResponse, error) {
	if s.commStore == nil {
		return nil, fmt.Errorf("server communication store not available")
	}
	return s.commStore.List(ServerCommFilter{
		TaskID:    taskID,
		ModelName: modelName,
		Page:      page,
		Limit:     limit,
	})
}

func (s *Service) GetServerCommunication(id string) (*ServerCommunication, error) {
	if s.commStore == nil {
		return nil, fmt.Errorf("server communication store not available")
	}
	return s.commStore.GetByID(id)
}

// GallerySyncContent resolves non-text content items for gallery models.
// For each unsynced asset it syncs the file to the model's gallery.
// If the file belongs to a character, it syncs the entire character as a group.
func (s *Service) GallerySyncContent(items []ContentItem, modelName string) ([]ContentItem, error) {
	log.Printf("[gallery-sync] GallerySyncContent start model=%q items=%d", modelName, len(items))

	if s.assetSyncStore == nil {
		log.Printf("[gallery-sync] assetSyncStore not available")
		return nil, fmt.Errorf("asset sync store not available")
	}

	m, err := s.providerStore.GetModelByName(modelName)
	if err != nil {
		log.Printf("[gallery-sync] GetModelByName error: %v", err)
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	if m == nil {
		log.Printf("[gallery-sync] model %q not found in DB", modelName)
		return nil, fmt.Errorf("model not found")
	}

	if ak, sk, _ := s.effectiveCredentials(m); ak == "" || sk == "" {
		log.Printf("[gallery-sync] no AK/SK for model %q (db_ak=%q env_ak=%q)", modelName, m.AccessKeyID, s.assetAccessKeyID)
		return nil, fmt.Errorf("no AK/SK configured for gallery sync. Set ASSET_ACCESS_KEY_ID / ASSET_SECRET_ACCESS_KEY env vars")
	}
	log.Printf("[gallery-sync] AK/SK OK for model %q, processing %d items", modelName, len(items))

	result := make([]ContentItem, len(items))
	copy(result, items)

	for i, item := range items {
		if item.Type == "text" || item.ID == "" {
			continue
		}
		log.Printf("[gallery-sync] processing item[%d] id=%q name=%q type=%q", i, item.ID, item.Name, item.Type)

		// Already synced?
		synced, err := s.assetSyncStore.GetByModelAndFile(m.ID, item.ID)
		if err == nil && synced != nil && synced.Status == "active" && synced.AssetID != "" {
			log.Printf("[gallery-sync] item[%d] already synced asset_id=%s", i, synced.AssetID)
			result[i].DataURL = "asset://" + synced.AssetID
			continue
		}
		log.Printf("[gallery-sync] item[%d] not synced yet, checking character linkage", i)

		// Not synced — check if file belongs to a character
		charIDs, cErr := s.charService.FindCharactersByFileID(item.ID)
		charSynced := false
		if cErr == nil && len(charIDs) > 0 {
			log.Printf("[gallery-sync] item[%d] belongs to characters %v, syncing character assets", i, charIDs)
			for _, charID := range charIDs {
				if _, syncErr := s.SyncCharacterAssets(&SyncCharacterRequest{
					ModelID: m.ID,
					CharacterID: charID,
				}); syncErr != nil {
					log.Printf("[gallery-sync] item[%d] SyncCharacterAssets(%s) error: %v", i, charID, syncErr)
					continue
				}
				// Re-check after sync
				synced2, _ := s.assetSyncStore.GetByModelAndFile(m.ID, item.ID)
				if synced2 != nil && synced2.Status == "active" && synced2.AssetID != "" {
					log.Printf("[gallery-sync] item[%d] synced via character %s asset_id=%s", i, charID, synced2.AssetID)
					result[i].DataURL = "asset://" + synced2.AssetID
					charSynced = true
					break
				}
			}
		} else {
			log.Printf("[gallery-sync] item[%d] no character linkage (cErr=%v)", i, cErr)
		}

		if !charSynced {
			log.Printf("[gallery-sync] item[%d] syncing as single file", i)
			resp, err := s.SyncAsset(&SyncAssetRequest{
				ModelID: m.ID,
				FileID:  item.ID,
			})
			if err == nil && resp.Status == "active" && resp.AssetID != "" {
				log.Printf("[gallery-sync] item[%d] single file sync OK asset_id=%s", i, resp.AssetID)
				result[i].DataURL = "asset://" + resp.AssetID
			} else {
				log.Printf("[gallery-sync] item[%d] single file sync FAILED err=%v status=%q", i, err, resp.Status)
			}
		}
	}

	log.Printf("[gallery-sync] GallerySyncContent done model=%q items=%d", modelName, len(items))
	return result, nil
}
