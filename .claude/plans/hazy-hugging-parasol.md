# New Studio — Model-Driven Generation Architecture

## Context
The current `internal/studio` manages API keys, presets, and defaults via JSON files. The user wants everything in the database (models table already exists in `internal/provider`) and a plugin-based generation system where each model type can have its own payload builder and response parser. The old studio goes to `studio-copy` for reference.

## Changes

### 1. Config (`config/config.go`)
- Remove `KeysFile`, `PresetsFile`, `DefaultModel` fields

### 2. Rename studio (`git mv internal/studio internal/studio-copy`)
- Preserves old code as reference
- All old routes remain broken until re-wired

### 3. New `internal/studio/` — Model Handler Plugin System

**`types.go`:**
```go
// GenerateRequest — input for POST /studio/generate
type GenerateRequest struct {
    ModelID string `json:"model_id" binding:"required"`
    Prompt  string `json:"prompt"`
    // common params that apply across models
    Duration    float64       `json:"duration"`
    AspectRatio string        `json:"aspect_ratio"`
    Refs        []MediaRef    `json:"refs"`
    // model-specific opaque params
    Params map[string]interface{} `json:"params"`
}

type MediaRef struct {
    Type string `json:"type"` // image, video, audio
    URL  string `json:"url"`
}

type GenerateResponse struct {
    TaskID  string `json:"task_id"`
    Status  string `json:"status"`
    ModelID string `json:"model_id"`
}

type StatusResult struct {
    Status   string `json:"status"`
    VideoURL string `json:"video_url,omitempty"`
    ImageURL string `json:"image_url,omitempty"`
    Error    string `json:"error,omitempty"`
}

// ModelHandler — each model type implements this
type ModelHandler interface {
    // Name returns a unique identifier for this handler
    Name() string
    // BuildPayload constructs the API-specific request body using the model config + request
    BuildPayload(model *provider.Model, req *GenerateRequest) ([]byte, error)
    // ParseResponse extracts task ID from the API response
    ParseGenerateResponse(body []byte) (string, error)
    // ParseStatus extracts status + result URL from poll response
    ParseStatusResponse(body []byte) (*StatusResult, error)
    // GetStatusURL returns the API path for status polling
    GetStatusURL(taskID string) string
}

// HandlerFactory — registry of model handlers
type HandlerRegistry struct {
    handlers map[string]ModelHandler
}
```

**`service.go`:**
- `NewService(providerStore, db, outputsDir)`
- Registry of ModelHandlers (Seedance, Seedream, etc.)
- `Generate(modelID, req)` → looks up provider.Model → picks handler → builds payload → sends HTTP request → returns task ID
- `GetStatus(taskID, modelID)` → polls API via handler
- `CancelTask(taskID, modelID)` → cancels via handler
- Output file downloading (reuse from old studio)

**`handler.go`:**
- `POST /api/v1/studio/generate` → calls service.Generate
- `GET /api/v1/studio/status/:taskId` → calls service.GetStatus (needs model_id as query param or stored in-memory)
- `DELETE /api/v1/studio/task/:taskId` → calls service.CancelTask

**`seedance_handler.go`** (first handler):
- Implements ModelHandler for BytePlus Seedance models
- `BuildPayload`: same logic as old `buildPayload()` + old `compilePromptText()` + reference images
- `ParseGenerateResponse`: extracts `task_id` from BytePlus response
- `ParseStatusResponse`: extracts status + video URL from BytePlus response
- Auth: Bearer token from model.APIKey
- Sends to `model.URL + model.Endpoint` (e.g. `https://ark.ap-southeast.bytepluses.com/api/v3/contents/generations/tasks`)

**`seedream_handler.go`** (second handler):
- Implements ModelHandler for BytePlus Seedream models
- `BuildPayload`: builds Seedream-specific payload
- `ParseGenerateResponse`: extracts task ID
- `ParseStatusResponse`: extracts status + image URL
- Auth: Bearer token from model.APIKey

### 4. main.go
- Remove `cfg.KeysFile`, `cfg.PresetsFile`, `cfg.DefaultModel` references
- Wire new studio with provider store + DB
- Update routes

### 5. OpenAPI
- Update studio endpoints to use model-based generation
- Remove old key/preset endpoints
- Keep status, cancel, health endpoints

## Task List
1. Remove config fields (KeysFile, PresetsFile, DefaultModel)
2. git mv internal/studio → internal/studio-copy
3. Create new internal/studio/ types.go, service.go, handler.go, seedance_handler.go, seedream_handler.go
4. Update main.go wiring
5. Update OpenAPI docs
6. Update Dockerfile if needed
