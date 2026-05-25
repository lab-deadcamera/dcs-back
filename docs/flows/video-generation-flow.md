# Video Generation Flow

## Overview

```
Director assigns resources → User generates in Studio → Toma created → Director approves final take
```

The flow spans three roles:
- **Director** — configures the scene (presets, characters, assets) and selects the final approved take.
- **Studio User** — generates videos using only the resources the director assigned.
- **Backend** — orchestrates generation, stores outputs, and tracks take state.

---

## 1. Scene Configuration (Director)

The director pre-configures each scene before any generation happens:

### 1.1 Resource Assignment

**Endpoint:** `POST /api/v1/projects/:id/scenes/:sceneId/assignments/{presets,characters,assets}`

The director assigns:
- **Presets** — which lens, camera, motion, color grading, and genre presets are available.
- **Characters** — which characters (and their files) can be used as reference.
- **Assets** — which uploaded files (images, video, audio) can be used as reference.

### 1.2 Scene Lock

Once the director finalizes the scene configuration, the scene can be **locked** (future feature). Locked scenes prevent further resource changes until unlocked.

---

## 2. Generation (Studio User)

### 2.1 Session Start

The user opens the Studio and selects a project/scene via the session gate dialog.

**On session init:**
1. `GET /api/v1/projects/:id/scenes/:sceneId/assignments` — loads assigned resources.
2. StudioStore stores:
   - `scenePresetIds` — filters available cinematography presets.
   - `sceneCharacterIds` — filters available characters in the asset panel.
   - `sceneAssetIds` — filters available files.
   - `freeAssets` — assigned files loaded as reference assets.

### 2.2 Generation Request

**Endpoint:** `POST /api/v1/studio/video/generate`

Payload includes:
```json
{
  "model": "dreamina-seedance-2-0-260128",
  "content": [
    { "type": "text", "text": "prompt description" },
    { "type": "image", "id": "file-uuid", "name": "ref.png" }
  ],
  "project_id": "...",
  "scene_id": "...",
  "scene_code": "SC01",
  "take_number": 3,
  "user_id": 1
}
```

**Flow:**
1. Backend validates project/scene/take context.
2. Creates a `GenerationLog` entry via defer (always saved, even on failure).
3. Resolves content (file IDs → public URLs or asset:// URIs if synced).
4. Picks the matching generator (Seedance, Seedream, etc.).
5. Calls the external AI API.
6. Tracks the task in memory for status polling.
7. Estimates cost via `CostCalculator`.
8. Returns `{ taskId, status: "running" }`.

### 2.3 Status Polling

**Endpoint:** `GET /api/v1/studio/video/status/:taskId`

The frontend polls every 2-3 seconds until the status is `"succeeded"` or `"failed"`.

**On success:**
1. Generator downloads the video from the external API.
2. Saves locally to `{OUTPUTS_DIR}/{scene}_T{take}_U{user}_{datetime}.mp4`.
3. Updates `GenerationLog` with final status and outputs.
4. Creates `GeneratedAsset` records.
5. Creates a take record in the scene.

### 2.4 Local Filename Pattern

When a generation succeeds, the backend renames the downloaded file:

```
{SceneCode}_T{TakeNumber}_U{UserID}_{YYYYMMDD}_{HHmmss}.mp4
```

Example:
```
SC01_T3_U1_20260525_143022.mp4
```

The frontend prefers the local URL (`/outputs/...`) over the external model URL when rendering the video in the viewer/download dialog.

---

## 3. Take Management

### 3.1 Take Creation

Each generation produces a **take** linked to the scene+number. Multiple takes can exist for the same scene+number pair, but only one can be **active** at a time.

### 3.2 Save Generation

**Endpoint:** `POST /api/v1/projects/:id/scenes/:sceneId/takes/save-generation`

When a generation completes:
1. The frontend calls `save-generation`.
2. If an active take with the same number already exists, it's **discarded** (`active = false`).
3. The new take is created as `active = true`.
4. The partial unique index ensures only one active take per `(scene_id, number)`.

### 3.3 Toggle Active

**Endpoint:** `POST /api/v1/projects/:id/scenes/:sceneId/takes/:takeId/toggle-active`

The director can:
1. Reactivate a discarded take (deactivates the currently active one).
2. This allows comparing different generated versions.

---

## 4. Director Final Take Selection (TODO)

### 4.1 Current State

Takes are managed via `active` flag. The active take is the one shown in the Studio and used for exports.

### 4.2 Future: Final Selection Workflow

The director will have a dedicated UI to:
1. View all takes for a scene (active + discarded).
2. Compare takes side by side.
3. Mark a take as **"final"** (approved for delivery).
4. Once a take is marked as final:
   - It becomes pinned (cannot be discarded by new generations).
   - It gets a visual badge in the timeline/studio.
   - A `final` column or `finalized_at` timestamp is set on the takes table.

### 4.3 Proposed DB Change

```sql
ALTER TABLE takes ADD COLUMN final BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE takes ADD COLUMN finalized_at TIMESTAMPTZ DEFAULT NULL;
CREATE INDEX idx_takes_final ON takes(scene_id) WHERE final = true;
```

### 4.4 Proposed Endpoints

| Method | Route | Description |
|--------|-------|-------------|
| `PATCH` | `/projects/:id/scenes/:sceneId/takes/:takeId/finalize` | Mark take as final |
| `PATCH` | `/projects/:id/scenes/:sceneId/takes/:takeId/unfinalize` | Unmark take as final |
| `GET` | `/projects/:id/scenes/:sceneId/takes?final=true` | Filter final takes only |

---

## 5. Logging & Cost Tracking

### 5.1 Generation Logs

Every generation is logged in `generation_logs` with:
- `resource_type` — `"video"`, `"image"`, `"audio"`, `"text"`.
- `content_types` — types of content sent (e.g. `"text,image"`).
- `estimated_cost` — USD cost from the CostCalculator.
- `cost_source` — `"api_response"`, `"calculator"`, `"pending"`.

### 5.2 Server Communications

Each API call to external AI providers is logged in `server_communications` for debugging and cost analysis.

### 5.3 Cost Calculation

Per-model cost calculators implement `CostCalculator` interface:
- **Seedance**: Token-based formula `unitPrice / 1M × tokens × W × H × FPS / 1024`.
- **Seedream**: Fixed price per image (`$0.04/image`).
- **Gemini**: Free.

---

## 6. Resource Filtering in Studio

### 6.1 Presets

The cinematography panel only shows presets whose IDs are in `scenePresetIds`.

### 6.2 Characters

The character assets panel only shows characters whose IDs are in `sceneCharacterIds`.

### 6.3 Files/Assets

The free assets panel is populated from the `assets` array of the assignments endpoint.
