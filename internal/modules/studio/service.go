package studio

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"dcs-back-v0/config"
	"dcs-back-v0/internal/modules/character"
	"dcs-back-v0/internal/modules/file"
	"dcs-back-v0/internal/modules/provider"
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

	type ShotLookup func(sceneID string) string
type Service struct {
	providerStore        *provider.Store
	fileService          *file.Service
	charService          *character.Service
	outputsDir           string
	pipelineGens         []PipelineRunner
	costCalcs            []CostCalculator
	tasks                map[string]*TaskRecord
	assetSyncStore       *AssetSyncStore
	baseURL              string
	logStore             *GenerationLogStore
	commStore            *ServerCommunicationStore
	assetStore           *GeneratedAssetStore
	takeSaver            TakeSaver
	shotLookup           ShotLookup
	assetAccessKeyID     string
	assetSecretAccessKey string
	assetDefaultGroupID  string
	assetAutoNormalize   bool
	assetAspectFix       string
	assetAIRepair        bool
	assetImageModel      string
	mu                   sync.RWMutex
}

func (s *Service) GetProviderStore() *provider.Store {
	return s.providerStore
}

func NewService(providerStore *provider.Store, fileService *file.Service, outputsDir, baseURL string) *Service {
	return &Service{
		providerStore: providerStore,
		fileService:   fileService,
		outputsDir:    outputsDir,
		baseURL:       baseURL,
		pipelineGens:  []PipelineRunner{},
		costCalcs:     []CostCalculator{},
		tasks:         make(map[string]*TaskRecord),
	}
}

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

// SetAssetFixOptions configures the automatic repair of assets that fail
// BytePlus validation (aspect ratio / height limits).
func (s *Service) SetAssetFixOptions(autoNormalize bool, aspectFix string, aiRepair bool, imageModel string) {
	s.assetAutoNormalize = autoNormalize
	s.assetAspectFix = aspectFix
	s.assetAIRepair = aiRepair
	s.assetImageModel = imageModel
}

func (s *Service) SetTakeSaver(saver TakeSaver) {
	s.takeSaver = saver

	// ShotLookup resolves a shot ID from a scene ID for legacy generation logs.

}

	func (s *Service) SetShotLookup(lookup ShotLookup) {
		s.shotLookup = lookup
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

func (s *Service) RegisterCalculator(calc CostCalculator) {
	s.costCalcs = append(s.costCalcs, calc)
}

func (s *Service) pickCalculator(modelName string) CostCalculator {
	for _, c := range s.costCalcs {
		if c.Match(modelName) {
			return c
		}
	}
	return nil
}

func (s *Service) GenerateUnified(req *StudioGenerateRequest) (*StudioGenerateResponse, error) {
	// Validar que los campos de sesiÃƒÆ’Ã‚Â³n estÃƒÆ’Ã‚Â©n presentes (obligatorios para tracking).
	if req.ProjectID == "" || req.SceneID == "" || req.SceneCode == "" || req.TakeNumber <= 0 {
		return nil, fmt.Errorf("project_id, scene_id, scene_code and take_number are required for generation")
	}

	var (
		genReq        *GeneratorRequest
		modelName     string
		taskID        string
		status        = "failed"
		outputs       []OutputResource
		errLog        string
		estimatedCost float64
		costSource    string
	)

	// Defer log save ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â runs on every return path (including early errors)
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
			ProjectName:   req.ProjectName,
			SceneID:       req.SceneID,
			ShotID:        req.ShotID,
			SceneCode:     req.SceneCode,
			TakeNumber:    req.TakeNumber,
			Request:       string(reqBytes),
			Outputs:       outputs,
			Status:        status,
			ErrorMessage:  errLog,
			ResourceType:  req.ResourceType,
			ContentTypes:  extractContentTypes(req.Content),
			EstimatedCost: estimatedCost,
			CostSource:    costSource,
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
	// Compute total input video duration from resolved content
	inputDuration := float64(0)
	for _, item := range resolvedContent {
		if item.Type == "video" && item.ID != "" {
			if f, err := s.fileService.GetFile(item.ID); err == nil && f != nil {
				inputDuration += f.Duration
			}
		}
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

	// Build the actual API payload (for logging and server communications)
	apiPayload := gen.BuildPayload(genReq)
	apiPayloadBytes, _ := json.Marshal(apiPayload)

	genStart := time.Now()
	result, err := gen.Generate(genReq)
	genDur := time.Since(genStart).Milliseconds()

	// Log server communication with the actual request body
	if s.commStore != nil {
		respBody := ""
		genStatus := 200
		if err != nil {
			genStatus = 0
			respBody = err.Error()
		} else if result != nil && result.Raw != nil {
			rawBytes, _ := json.Marshal(result.Raw)
			respBody = string(rawBytes)
		}
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		s.commStore.Create(&ServerCommunication{
			ModelName:    m.Name,
			Endpoint:     m.URL + m.Endpoint,
			Method:       "POST",
			RequestBody:  string(apiPayloadBytes),
			ResponseBody: respBody,
			StatusCode:   genStatus,
			DurationMs:   genDur,
			ErrorMessage: errMsg,
		})
	}
	log.Printf("[generate-unified] gen.Generate dur=%dms err=%v", genDur, err != nil)

	// aiCall already set above from BuildPayload
	if result != nil {
		taskID = result.TaskID
		status = result.Status
		if len(result.Outputs) > 0 {
			outputs = result.Outputs
		}
	}
	if err != nil {
		errLog = err.Error()
		return nil, err
	}
	// Calculate estimated cost
	if calc := s.pickCalculator(modelName); calc != nil {
		if cost, ok := calc.CalculateFromResponse(result.Raw, genReq); ok {
			estimatedCost = cost
			costSource = "api_response"
		} else if !calc.NeedsBackgroundCalc() {
			estimatedCost = calc.CalculateEstimated(genReq)
			costSource = "calculator"
		} else {
			costSource = "pending"
			// Background calculation handled separately
		}
	}
	// Store naming info in task record for local filename
	userName := req.UserName
	if userName == "" {
		userName = fmt.Sprintf("u%d", req.UserID)
	}

	// Track the task for status polling
	s.mu.Lock()
	s.tasks[result.TaskID] = &TaskRecord{
		TaskID:      result.TaskID,
		ModelID:     m.ID,
		ModelName:   m.Name,
		Status:      result.Status,
		ProjectName: req.ProjectName,
		SceneCode:   req.SceneCode,
		TakeNumber:  req.TakeNumber,
		UserHandle:  userName,
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
		AssetType:    strings.ToUpper(detectAssetType(f.MimeType)),
	}
	if err := s.assetSyncStore.Create(record); err != nil {
		log.Printf("[sync-asset] Create sync record error: %v", err)
		return nil, fmt.Errorf("failed to create sync record: %w", err)
	}
	log.Printf("[sync-asset] sync record created id=%s", record.ID)

	// Build the publicly accessible URL for the file
	fileURL := s.baseURL + "/api/v1/files/" + req.FileID + "/serve"
	uploadName := f.Filename
	normalized := false

	// Auto-normalize images that violate BytePlus dimension limits (aspect
	// ratio / height), uploading a derived copy so the original file is
	// never modified and model_assets.file_id keeps pointing at it.
	if s.assetAutoNormalize && s.fileService != nil && isImageMime(f.MimeType) {
		if servePath, err := s.fileService.GetServePath(req.FileID); err == nil {
			if data, err := normalizeImage(servePath, s.assetAspectFix); err != nil {
				log.Printf("[sync-asset] normalizeImage error: %v", err)
			} else if data != nil {
				up, upErr := s.fileService.Upload(data, "normalized-"+req.FileID+".jpg", "temp", "temp")
				if upErr != nil || up == nil {
					log.Printf("[sync-asset] upload normalized copy failed: %v", upErr)
				} else {
					fileURL = s.baseURL + "/api/v1/files/" + up.ID + "/serve"
					uploadName = "normalized-" + f.Filename
					normalized = true
					log.Printf("[sync-asset] normalized image %q -> derived file %s", req.FileID, up.ID)
				}
			}
		}
	}

	// Upload to the asset library
	log.Printf("[sync-asset] calling CreateAsset url=%q filename=%q type=%q normalized=%v", fileURL, uploadName, detectAssetType(f.MimeType), normalized)
	api := NewAssetAPI(ak, sk, groupID)
	api.SetCommStore(s.commStore)
	result, err := api.CreateAsset(fileURL, uploadName, detectAssetType(f.MimeType), "")
	if err != nil {
		log.Printf("[sync-asset] CreateAsset FAILED: %v", err)
		s.assetSyncStore.UpdateStatus(record.ID, "failed", err.Error(), "", "", "", "")
		return &SyncAssetResponse{
			ID:           record.ID,
			ModelID:      req.ModelID,
			FileID:       req.FileID,
			Status:       "failed",
			ErrorMessage: err.Error(),
			Normalized:   normalized,
		}, nil
	}

	assetID, _ = result["id"].(string)
	record.AssetID = assetID
	log.Printf("[sync-asset] CreateAsset OK asset_id=%s", assetID)

	// Poll until Active (up to ~2 min)
	log.Printf("[sync-asset] polling asset %s for Active status", assetID)
	assetStatus := ""
	assetURL := ""
	assetType := ""
	referenceURI := ""
	for i := 0; i < 20; i++ {
		statusResult, err := api.GetAsset(assetID, "")
		if err != nil {
			log.Printf("[sync-asset] poll[%d] GetAsset error: %v", i, err)
			time.Sleep(3 * time.Second)
			continue
		}
		assetStatus, _ = statusResult["Status"].(string)
		if url, ok := statusResult["URL"].(string); ok && url != "" {
			assetURL = url
		}
		if at, ok := statusResult["AssetType"].(string); ok && at != "" {
			assetType = strings.ToUpper(at)
		}
		log.Printf("[sync-asset] poll[%d] status=%q url_set=%v", i, assetStatus, assetURL != "")
		if assetStatus == "Active" || assetStatus == "Failed" {
			break
		}
		time.Sleep(3 * time.Second)
	}

	// Construir la URI de referencia segÃƒÂºn el tipo de modelo
	referenceURI = BuildReferenceURI(m.Name, assetID, assetURL)

	finalStatus := "active"
	errMsg := ""
	if assetStatus != "Active" {
		finalStatus = "failed"
		errMsg = fmt.Sprintf("asset did not become Active, last status: %s", assetStatus)
		log.Printf("[sync-asset] final status NOT Active: %q", assetStatus)
	} else {
		log.Printf("[sync-asset] asset is now Active url=%s", assetURL)
	}

	// Update the record
	if err := s.assetSyncStore.UpdateStatus(record.ID, finalStatus, errMsg, assetID, assetURL, assetType, referenceURI); err != nil {
		return nil, fmt.Errorf("failed to update sync status: %w", err)
	}

	// Also update the in-memory record
	record.Status = finalStatus
	record.ErrorMessage = errMsg
	record.AssetURL = assetURL
	record.AssetType = assetType
	record.ReferenceURI = referenceURI

	log.Printf("[sync-asset] SyncAsset done record_id=%s asset_id=%s status=%s", record.ID, assetID, finalStatus)
	return &SyncAssetResponse{
		ID:           record.ID,
		ModelID:      req.ModelID,
		FileID:       req.FileID,
		AssetID:      assetID,
		AssetGroupID: groupID,
		Status:       finalStatus,
		ErrorMessage: errMsg,
		ReferenceURI: referenceURI,
		Normalized:   normalized,
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

// ListGalleryModels returns the models that have records in model_assets,
// with per-status counts — powers the admin "External Galleries" view.
func (s *Service) ListGalleryModels() ([]GalleryModel, error) {
	if s.assetSyncStore == nil {
		return nil, fmt.Errorf("asset sync store not available")
	}
	summaries, err := s.assetSyncStore.ListModelSummaries()
	if err != nil {
		return nil, err
	}

	models := make([]GalleryModel, 0, len(summaries))
	for _, sm := range summaries {
		name := "unknown"
		if s.providerStore != nil {
			if m, err := s.providerStore.GetModelByID(sm.ModelID); err == nil && m != nil {
				name = m.Name
			}
		}
		models = append(models, GalleryModel{
			ModelID:   sm.ModelID,
			ModelName: name,
			Total:     sm.Total,
			Active:    sm.Active,
			Failed:    sm.Failed,
			Syncing:   sm.Syncing,
			LastSync:  sm.LastSync,
		})
	}
	return models, nil
}

// ListGalleryAssets returns a model's sync records enriched with the internal
// gallery match (file name/mime) and the characters that reference each file.
func (s *Service) ListGalleryAssets(modelID string) ([]GalleryAsset, error) {
	if s.assetSyncStore == nil {
		return nil, fmt.Errorf("asset sync store not available")
	}
	records, err := s.assetSyncStore.ListByModel(modelID)
	if err != nil {
		return nil, err
	}

	assets := make([]GalleryAsset, 0, len(records))
	for _, ma := range records {
		asset := GalleryAsset{ModelAsset: ma}
		if s.fileService != nil {
			if f, err := s.fileService.GetFile(ma.FileID); err == nil && f != nil {
				asset.FileName = f.Filename
				asset.MimeType = f.MimeType
			}
		}
		if s.charService != nil {
			if ids, err := s.charService.FindCharactersByFileID(ma.FileID); err == nil {
				for _, id := range ids {
					if c, err := s.charService.GetByID(id); err == nil && c != nil {
						asset.Characters = append(asset.Characters, GalleryCharacter{ID: c.ID, Name: c.Name})
					}
				}
			}
		}
		if asset.Characters == nil {
			asset.Characters = []GalleryCharacter{}
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

// ListGalleryErrors returns the last 5 failed sync attempts for a specific
// file in a model.
func (s *Service) ListGalleryErrors(modelID, fileID string) ([]ModelAsset, error) {
	if s.assetSyncStore == nil {
		return nil, fmt.Errorf("asset sync store not available")
	}
	return s.assetSyncStore.ListRecentErrors(modelID, fileID, 5)
}

// RepairImageWithAI regenerates an image with an image model, using the
// failing file as a reference (same pipeline as app-image-gen-panel), then
// overwrites the original file in place — respecting its path and regenerating
// its thumbnail — and returns the same file id. `modelName` selects the
// generator model; when empty it falls back to the configured ASSET_IMAGE_MODEL.
func (s *Service) RepairImageWithAI(fileID, prompt, ratio, resolution, modelName string) (string, error) {
	resolveName := modelName
	if resolveName == "" {
		resolveName = s.assetImageModel
	}
	if resolveName == "" {
		return "", fmt.Errorf("no image model selected for AI repair (choose one in the dialog or set ASSET_IMAGE_MODEL)")
	}
	if s.fileService == nil || s.providerStore == nil {
		return "", fmt.Errorf("file or provider service not available")
	}

	f, err := s.fileService.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}
	if f == nil {
		return "", fmt.Errorf("file not found")
	}
	if !isImageMime(f.MimeType) {
		return "", fmt.Errorf("file is not an image; AI repair is only for images")
	}

	m, gen, err := s.resolveImageModel(resolveName)
	if err != nil {
		return "", err
	}

	if prompt == "" {
		prompt = f.Filename
	}
	req := &GeneratorRequest{
		Model:      m.Name,
		Content: []ContentItem{
			{Type: "image", DataURL: s.baseURL + "/api/v1/files/" + fileID + "/serve", Name: f.Filename},
			{Type: "text", Text: prompt},
		},
		Ratio:      ratio,
		Resolution: resolution,
		APIKey:     m.APIKey,
		BaseURL:    m.URL,
		Endpoint:   m.Endpoint,
	}

	result, err := gen.Generate(req)
	if err != nil {
		return "", fmt.Errorf("AI generation failed: %w", err)
	}

	outputURL := ""
	if len(result.Outputs) > 0 {
		outputURL = result.Outputs[0].URL
	}

	// Async generators: poll until terminal (up to ~60s).
	if outputURL == "" && result.TaskID != "" && result.Status != config.STATUS_SUCCESS && result.Status != config.STATUS_FAILED {
		for i := 0; i < 20; i++ {
			time.Sleep(3 * time.Second)
			st, err := gen.GetStatus(result.TaskID, m.APIKey, m.URL, m.Endpoint)
			if err != nil {
				continue
			}
			if st.Status == config.STATUS_SUCCESS && len(st.Outputs) > 0 {
				outputURL = st.Outputs[0].URL
				break
			}
			if st.Status == config.STATUS_FAILED {
				return "", fmt.Errorf("AI repair failed: %s", st.Error)
			}
		}
	}
	if outputURL == "" {
		return "", fmt.Errorf("AI repair produced no image")
	}

	data, err := downloadBytes(resolveOutputURL(outputURL, s.baseURL))
	if err != nil {
		return "", fmt.Errorf("failed to download generated image: %w", err)
	}

	// Overwrite the original file in place so its path, URL and thumbnail stay
	// stable — the gallery and any references keep pointing at the same file.
	if _, err := s.fileService.ReplaceImageContent(fileID, data); err != nil {
		return "", fmt.Errorf("failed to store generated image: %w", err)
	}
	log.Printf("[fix-asset] AI repair file=%q replaced in place url=%q", fileID, outputURL)
	return fileID, nil
}

// FixAsset retries a failed asset sync, optionally repairing the image first:
//   - mode "ai"        → regenerate with the image generator, then sync.
//   - mode "normalize" → sync (SyncAsset auto-normalizes geometry).
//   - mode "auto"      → sync; if it still fails and AI repair is enabled,
//     fall back to regenerating with AI and syncing the result.
func (s *Service) FixAsset(req *FixAssetRequest) (*FixAssetResult, error) {
	mode := req.Mode
	if mode == "" {
		mode = "auto"
	}

	switch mode {
	case "ai":
		newFileID, err := s.RepairImageWithAI(req.FileID, "", req.Ratio, "", req.Model)
		if err != nil {
			return &FixAssetResult{FileID: req.FileID, Status: "failed", ErrorMessage: err.Error(), UsedFix: "ai"}, nil
		}
		return s.syncFixed(req.ModelID, newFileID, "ai")

	case "normalize":
		res, err := s.SyncAsset(&SyncAssetRequest{ModelID: req.ModelID, FileID: req.FileID})
		if err != nil {
			return nil, err
		}
		usedFix := "none"
		if res.Normalized {
			usedFix = "normalize"
		}
		return &FixAssetResult{FileID: req.FileID, Status: res.Status, ErrorMessage: res.ErrorMessage, UsedFix: usedFix}, nil

	default: // auto
		res, err := s.SyncAsset(&SyncAssetRequest{ModelID: req.ModelID, FileID: req.FileID})
		if err != nil {
			return nil, err
		}
		if res.Status != "failed" {
			usedFix := "none"
			if res.Normalized {
				usedFix = "normalize"
			}
			return &FixAssetResult{FileID: req.FileID, Status: res.Status, ErrorMessage: res.ErrorMessage, UsedFix: usedFix}, nil
		}
		if !s.assetAIRepair {
			return &FixAssetResult{FileID: req.FileID, Status: res.Status, ErrorMessage: res.ErrorMessage, UsedFix: "none"}, nil
		}
		newFileID, err := s.RepairImageWithAI(req.FileID, "", req.Ratio, "", req.Model)
		if err != nil {
			return &FixAssetResult{FileID: req.FileID, Status: res.Status, ErrorMessage: res.ErrorMessage + " (AI repair: " + err.Error() + ")", UsedFix: "none"}, nil
		}
		return s.syncFixed(req.ModelID, newFileID, "ai")
	}
}

func (s *Service) syncFixed(modelID, fileID, usedFix string) (*FixAssetResult, error) {
	res, err := s.SyncAsset(&SyncAssetRequest{ModelID: modelID, FileID: fileID})
	if err != nil {
		return nil, err
	}
	return &FixAssetResult{FileID: fileID, Status: res.Status, ErrorMessage: res.ErrorMessage, UsedFix: usedFix}, nil
}

// resolveImageModel resolves the image model used by RepairImageWithAI.
// Resolution order:
//  1. exact name match on the configured model,
//  2. first image model whose name contains the configured name (BytePlus
//     models often carry version suffixes, e.g. "...-251224-260128"),
//  3. any image model that has a registered pipeline generator.
func (s *Service) resolveImageModel(configuredName string) (*provider.Model, PipelineRunner, error) {
	// 1. Exact match.
	if m, err := s.providerStore.GetModelByName(configuredName); err == nil && m != nil {
		if gen := s.pickGenerator(m.Name); gen != nil {
			return m, gen, nil
		}
	}

	models, err := s.providerStore.ListModels("image")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list image models: %w", err)
	}

	// 2. Partial (substring) match on the configured name.
	if configuredName != "" {
		for i := range models {
			if strings.Contains(models[i].Name, configuredName) {
				if gen := s.pickGenerator(models[i].Name); gen != nil {
					return &models[i].Model, gen, nil
				}
			}
		}
	}

	// 3. First usable image model.
	for i := range models {
		if gen := s.pickGenerator(models[i].Name); gen != nil {
			return &models[i].Model, gen, nil
		}
	}

	return nil, nil, fmt.Errorf("no usable image model found (configured %q not found and no image model with a generator)", configuredName)
}

// resolveOutputURL turns a generator output URL (possibly relative to the
// backend) into an absolute URL that can be fetched server-side.
func resolveOutputURL(url, baseURL string) string {
	if url == "" {
		return ""
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	if strings.HasPrefix(url, "/") {
		return baseURL + url
	}
	return baseURL + "/outputs/" + url
}

// downloadBytes fetches a URL and returns its body.
func downloadBytes(url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("empty url")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
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
	api.SetCommStore(s.commStore)

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
		api.SetCommStore(s.commStore)
	}

	// Process each file ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â skip if already synced, upload if new or failed
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
					ReferenceURI: a.ReferenceURI,
				})
				break
			}
		}
		if alreadySynced {
			continue
		}

		// Not synced or previously failed ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â upload
		log.Printf("[sync-char] uploading file %q to group %s", cf.FileID, groupID)
		r, err := s.uploadAndTrackAsset(req.ModelID, cf.FileID, groupID, m.Name, api)
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
func (s *Service) uploadAndTrackAsset(modelID, fileID, groupID, modelName string, api *AssetAPI) (*SyncAssetResponse, error) {
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
		AssetType:    strings.ToUpper(detectAssetType(f.MimeType)),
	}
	if err := s.assetSyncStore.Create(record); err != nil {
		return nil, fmt.Errorf("failed to create sync record: %w", err)
	}

	// Build the publicly accessible file URL
	fileURL := s.baseURL + "/api/v1/files/" + fileID + "/serve"

	// Upload to the asset library
	result, err := api.CreateAsset(fileURL, f.Filename, detectAssetType(f.MimeType), "")
	if err != nil {
		s.assetSyncStore.UpdateStatus(record.ID, "failed", err.Error(), "", "", "", "")
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
	assetURL := ""
	assetType := ""
	referenceURI := ""
	for i := 0; i < 20; i++ {
		statusResult, err := api.GetAsset(assetID, "")
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		assetStatus, _ = statusResult["Status"].(string)
		if url, ok := statusResult["URL"].(string); ok && url != "" {
			assetURL = url
		}
		if at, ok := statusResult["AssetType"].(string); ok && at != "" {
			assetType = strings.ToUpper(at)
		}
		if assetStatus == "Active" || assetStatus == "Failed" {
			break
		}
		time.Sleep(3 * time.Second)
	}

	// Construir la URI de referencia segÃƒÂºn el tipo de modelo
	referenceURI = BuildReferenceURI(modelName, assetID, assetURL)

	finalStatus := "active"
	errMsg := ""
	if assetStatus != "Active" {
		finalStatus = "failed"
		errMsg = fmt.Sprintf("asset did not become Active, last status: %s", assetStatus)
	}

	// Update the record
	if err := s.assetSyncStore.UpdateStatus(record.ID, finalStatus, errMsg, assetID, assetURL, assetType, referenceURI); err != nil {
		return nil, fmt.Errorf("failed to update sync status: %w", err)
	}

	record.Status = finalStatus
	record.ErrorMessage = errMsg
	record.AssetURL = assetURL
	record.AssetType = assetType
	record.ReferenceURI = referenceURI

	return &SyncAssetResponse{
		ID:           record.ID,
		ModelID:      modelID,
		FileID:       fileID,
		AssetID:      assetID,
		AssetGroupID: groupID,
		Status:       finalStatus,
		ErrorMessage: errMsg,
		ReferenceURI: referenceURI,
	}, nil
}

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

// statusFromLog recupera el estado de una tarea desde el log cuando
// la tarea ya no estÃƒÆ’Ã‚Â¡ en memoria (ej. reinicio del servidor).
func (s *Service) statusFromLog(log *GenerationLog) (*StatusResult, error) {
	// Si el log ya tiene estado final, devolver lo que tiene sin llamar al generador
	if log.Status == config.STATUS_SUCCESS || log.Status == config.STATUS_FAILED {
		sr := &StatusResult{
			Status: log.Status,
			Error:  log.ErrorMessage,
		}
		if len(log.Outputs) > 0 {
			sr.VideoURL = log.Outputs[0].URL
			sr.LocalURL = log.Outputs[0].LocalURL
		}
		return sr, nil
	}

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

	// Log server communication
	if s.commStore != nil {
		reqBytes, _ := json.Marshal(map[string]string{"task_id": log.TaskID})
		respBody := ""
		genStatus := 200
		if err != nil {
			genStatus = 0
			respBody = err.Error()
		} else if result != nil && result.Raw != nil {
			rawBytes, _ := json.Marshal(result.Raw)
			respBody = string(rawBytes)
		}
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		s.commStore.Create(&ServerCommunication{
			TaskID:       log.TaskID,
			ModelName:    m.Name,
			Endpoint:     m.URL + m.Endpoint,
			Method:       "GET",
			RequestBody:  string(reqBytes),
			ResponseBody: respBody,
			StatusCode:   genStatus,
			ErrorMessage: errMsg,
		})
	}
	fmt.Printf("[status-from-log] gen.GetStatus err=%v", err != nil)
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

	if result.Status == config.STATUS_SUCCESS || result.Status == config.STATUS_FAILED {
		s.updateLogWithFinalStatus(log.TaskID, result)
		if result.Status == config.STATUS_SUCCESS {
			s.saveGeneratedAssets(log.TaskID, result)
		}
	}

	return statusResult, nil
}

// saveToTakes persiste las URLs del output en la tabla takes cuando
// una generaciÃƒÂ³n de video se completa exitosamente.
// Luego actualiza el generation_log con los outputs para mantener
// la consistencia entre el log y los takes.
func (s *Service) saveToTakes(taskID string, videoURL, localURL string) {
	if s.takeSaver == nil || s.logStore == nil {
		return
	}
	log, logErr := s.logStore.GetByTaskID(taskID)
	if logErr != nil || log == nil {
		return
	}
	if log.SceneID == "" || log.TakeNumber <= 0 {
		return
	}

	if err := s.takeSaver(log.ShotID, log.TakeNumber, videoURL, localURL, taskID); err != nil {
		fmt.Printf("failed to save take for task %s: %v\n", taskID, err)
		return
	}

	// Actualizar el generation_log con los outputs (video URL/local URL)
	outputs := []OutputResource{{URL: videoURL, LocalURL: localURL, Type: "video"}}
	if err := s.logStore.UpdateByTaskID(taskID, outputs, log.Status, log.ErrorMessage); err != nil {
		fmt.Printf("failed to update generation log outputs for task %s: %v\n", taskID, err)
	}
}

// El generador de los modelos tiene la responsabilidad de guardar el video en los assets carpeta outputs/.
// Cuando se completa el proceso de generación.
func (s *Service) GetStatus(taskID string) (*StatusResult, error) {
	s.mu.RLock()
	record, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if !ok {
		log, logErr := s.logStore.GetByTaskID(taskID)
		if logErr != nil || log == nil {
			return nil, fmt.Errorf("unknown task: %s", taskID)
		}
		resp, err := s.statusFromLog(log)
		if err != nil {
			return nil, err
		}

		// Rename local file if the generator downloaded it
		if resp.Status == config.STATUS_SUCCESS && resp.LocalURL != "" {
			s.saveToTakes(taskID, resp.VideoURL, resp.LocalURL)
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

		// Log server communication
		if s.commStore != nil {
			reqBytes, _ := json.Marshal(map[string]string{"task_id": taskID})
			respBody := ""
			genStatus := 200
			if err != nil {
				genStatus = 0
				respBody = err.Error()
			} else if result != nil && result.Raw != nil {
				rawBytes, _ := json.Marshal(result.Raw)
				respBody = string(rawBytes)
			}
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			s.commStore.Create(&ServerCommunication{
				TaskID:       taskID,
				ModelName:    m.Name,
				Endpoint:     m.URL + m.Endpoint,
				Method:       "GET",
				RequestBody:  string(reqBytes),
				ResponseBody: respBody,
				StatusCode:   genStatus,
				ErrorMessage: errMsg,
			})
		}
		log.Printf("[get-status] gen.GetStatus err=%v", err != nil)
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

		if result.Status == config.STATUS_SUCCESS || result.Status == config.STATUS_FAILED {
			s.mu.Lock()
			record.Status = result.Status
			record.Result = statusResult
			s.mu.Unlock()
			// Update generation log with final AI response
			s.updateLogWithFinalStatus(taskID, result)
			if result.Status == config.STATUS_SUCCESS {
				s.saveGeneratedAssets(taskID, result)
				s.saveToTakes(taskID, statusResult.VideoURL, statusResult.LocalURL)
			}
		}
		return statusResult, nil
	}

	return nil, fmt.Errorf("no generator available for model: %s", m.Name)
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

	return fmt.Errorf("no generator available for model: %s", m.Name)
}

// ListGenerationLogs returns paginated generation logs, optionally filtered.
func (s *Service) ListGenerationLogs(page, limit int, projectID, sceneID, status, modelName string, userID int, dateFrom, dateTo, resourceType string) (*ListGenerationLogsResponse, error) {
	if s.logStore == nil {
		return nil, fmt.Errorf("log store not available")
	}

	var (
		logs  []GenerationLog
		total int
		err   error
	)
	if projectID != "" || sceneID != "" || status != "" || modelName != "" || userID > 0 || dateFrom != "" || dateTo != "" || resourceType != "" {
		logs, total, err = s.logStore.ListByFilter(page, limit, projectID, sceneID, status, modelName, userID, dateFrom, dateTo, resourceType)
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

// SumGenerationLogsCost returns the total estimated_cost matching the given filters (no pagination).
func (s *Service) SumGenerationLogsCost(projectID, sceneID, status, modelName string, userID int, dateFrom, dateTo, resourceType string) (*CostSummaryResponse, error) {
	if s.logStore == nil {
		return nil, fmt.Errorf("log store not available")
	}
	total, err := s.logStore.SumCostByFilter(projectID, sceneID, status, modelName, userID, dateFrom, dateTo, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to sum generation logs cost: %w", err)
	}
	return &CostSummaryResponse{TotalCost: total}, nil
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

// updateLogWithFinalStatus updates the generation log with the final AI response
// when an async task completes (succeeded or failed).
func (s *Service) updateLogWithFinalStatus(taskID string, result *GeneratorResult) {
	if s.logStore == nil {
		return
	}

	log, logErr := s.logStore.GetByTaskID(taskID)
	if logErr != nil || log == nil {
		return
	}

	errorMsg := result.Error

	if saveErr := s.logStore.UpdateByTaskID(taskID, result.Outputs, result.Status, errorMsg); saveErr != nil {
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
					// Usar la reference_uri especÃƒÂ­fica del modelo (asset:// para gallery, URL directa para otros)
					ci.DataURL = synced.ReferenceURI
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
			result[i].DataURL = synced.ReferenceURI
			continue
		}
		log.Printf("[gallery-sync] item[%d] not synced yet, checking character linkage", i)

		// Not synced ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â check if file belongs to a character
		charIDs, cErr := s.charService.FindCharactersByFileID(item.ID)
		charSynced := false
		if cErr == nil && len(charIDs) > 0 {
			log.Printf("[gallery-sync] item[%d] belongs to characters %v, syncing character assets", i, charIDs)
			for _, charID := range charIDs {
				if _, syncErr := s.SyncCharacterAssets(&SyncCharacterRequest{
					ModelID:     m.ID,
					CharacterID: charID,
				}); syncErr != nil {
					log.Printf("[gallery-sync] item[%d] SyncCharacterAssets(%s) error: %v", i, charID, syncErr)
					continue
				}
				// Re-check after sync
				synced2, _ := s.assetSyncStore.GetByModelAndFile(m.ID, item.ID)
				if synced2 != nil && synced2.Status == "active" && synced2.AssetID != "" {
					log.Printf("[gallery-sync] item[%d] synced via character %s asset_id=%s", i, charID, synced2.AssetID)
					result[i].DataURL = synced2.ReferenceURI
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
				result[i].DataURL = resp.ReferenceURI
			} else if err != nil {
				log.Printf("[gallery-sync] item[%d] single file sync FAILED err=%v", i, err)
			} else {
				log.Printf("[gallery-sync] item[%d] single file sync status=%q asset_id=%s", i, resp.Status, resp.AssetID)
			}
		}
	}

	log.Printf("[gallery-sync] GallerySyncContent done model=%q items=%d", modelName, len(items))
	return result, nil
}

// extractContentTypes returns a sorted, comma-separated list of unique content types.
func extractContentTypes(items []ContentItem) string {
	seen := make(map[string]bool)
	var types []string
	for _, item := range items {
		if item.Type != "" && !seen[item.Type] {
			seen[item.Type] = true
			types = append(types, item.Type)
		}
	}
	sort.Strings(types)
	if len(types) == 0 {
		return ""
	}
	return strings.Join(types, ",")
}

// renameOutputFile renames a locally-downloaded output file to follow the
// pattern: {ProjectName}_{SceneName}_T{take}_{user}_{datetime}.mp4
func (s *Service) renameOutputFile(localURL, projectName, sceneName string, takeNumber int, userHandle string) string {
	if localURL == "" || sceneName == "" {
		return ""
	}

	ext := ".mp4"
	now := time.Now()
	ts := now.Format("20060102_150405")

	safe := func(s string) string {
		r := strings.NewReplacer("/", "_", " ", "_", ":", "_")
		return r.Replace(s)
	}

	userPart := ""
	if userHandle != "" && userHandle != "0" && userHandle != "u0" {
		userPart = "_" + userHandle
	}

	prefix := safe(projectName)
	if prefix != "" {
		prefix += "_"
	}
	newName := fmt.Sprintf("%s%s_T%d%s_%s%s",
		prefix, safe(sceneName), takeNumber, userPart, ts, ext)
	oldPath := s.outputsDir + "/" + filepath.Base(localURL)
	newPath := s.outputsDir + "/" + newName

	if err := os.Rename(oldPath, newPath); err != nil {
		return ""
	}

	return config.OutPutUrl() + "/" + newName
}
