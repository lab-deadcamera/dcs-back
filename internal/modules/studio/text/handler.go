package text

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"dcs-back-v0/internal/modules/provider"
	skillmodule "dcs-back-v0/internal/modules/skill"
	"dcs-back-v0/internal/utils"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	providerStore *provider.Store
	skillSvc      *skillmodule.Service
}

func NewHandler(providerStore *provider.Store, skillSvc *skillmodule.Service) *Handler {
	return &Handler{providerStore: providerStore, skillSvc: skillSvc}
}

func (h *Handler) Generate(c *gin.Context) {
	utils.BadRequest(c, "text generation not yet implemented")
}

func (h *Handler) GetStatus(c *gin.Context) {
	utils.BadRequest(c, "text generation not yet implemented")
}

func (h *Handler) CancelTask(c *gin.Context) {
	utils.BadRequest(c, "text generation not yet implemented")
}

func (h *Handler) PreviewPayload(c *gin.Context) {
	utils.BadRequest(c, "text generation not yet implemented")
}

// ─── Default system prompts (fallbacks when no skill is selected) ─────

const defaultShotBuilderPrompt = `## Description
Single skill combining:
- **Dramatic / physical interpretation** (DCS-DIRECTION engine) — turns literary / metaphorical scripts into observable physical actions.
- **Visual composition and cinematography** (Cinema Worldbuilder Pro 2.0 engine) — builds cinematic scenes with compositional rigor.

Delivers a **Seedance Shotlist** with beats, duration, references, and a final prompt ready to run on Seedance / Higgsfield.

## Purpose
- Convert literary/metaphorical scripts into observable physical actions.
- Build cinematic scenes with compositional rigor.
- Deliver an artifact: **Seedance Shotlist · [Sequence] · [Mode]**

## Philosophy
- **Analog realism**: no plastic or commercial look.
- **Physical acting, not abstract**: emotions → transitive verbs → muscular actions.
- **Blocked composition**: every shot answers who is in frame, where, what they do, how the camera moves, and how it closes.

## Workflow
1. **Flexible input** — Abstract script → interpretation engine (Weston + Mamet). Physical shot → direct composition engine.
2. **Ambiguity Gate** — Resolve critical unknowns: subjects, references, location, duration, aspect ratio.
3. **Shot Breakdown** — Table with mood line → subtext → transitive verb → physical action → mode → runtime → references. User confirmation.
4. **Pre-prompt check** — Reference list, mode, scene, characters, Frame Map, camera, runtime.
5. **Final delivery (artifact)** — Numbered beats with title + duration. Full prompt in English with the 10 fixed blocks.

## Output format

Return ONLY a valid JSON object — NO markdown fences, NO comments, NO extra text:

{
  "description": "Wyatt & Mike - Fraternal conflict in the living room",
  "duration": 79,
  "mode": "M1",
  "aspectRatio": "9:16",
  "references": [
    { "slot": "@image1", "assetId": "wyatt", "type": "character" },
    { "slot": "@image2", "assetId": "mike", "type": "character" }
  ],
  "sequenceFlow": {
    "title": "Time budget",
    "subtitle": "Temperature rises with the conflict",
    "duration": 80,
    "metric": "dramaticIntensity",
    "scale": { "start": "Cold", "middle": "Hot", "end": "Empty" },
    "segments": [
      { "id": "A", "shotId": "A", "label": "Hook", "start": 0, "end": 11, "intensity": 0.2, "color": "#3d8b8f" }
    ]
  },
  "directorNotes": {
    "goal": "One-line director goal for this scene.",
    "styleGuide": "teal-amber grade - spherical rectilinear lens - flat field no vignette - 24fps 180 degree - diegetic audio only - prompt in positive",
    "warnings": ["Warning 1", "Warning 2"]
  },
  "shots": [
    {
      "id": "A",
      "title": "Mike bursts in + explodes",
      "description": "Wyatt enters and slams the door; Mike bursts in behind him yelling.",
      "duration": 11,
      "start": 0,
      "end": 11,
      "camera": {
        "lens": "40mm to 55mm",
        "framing": "Staggered two-shot to single Mike",
        "movement": "Handheld",
        "fps": 24,
        "shutter": "180 degree",
        "aspectRatio": "9:16"
      },
      "composition": {
        "frameMap": "Cut 1 (0-6s): Staggered two-shot. @image1 left third x=34%. @image2 right third x=70%. Cut 2 (6-11s): Single @image2.",
        "subjectLock": "@image1 + @image2: identical face/wardrobe. Shoulders square to camera.",
        "crossFrameRules": "@image1 left, @image2 right - never cross center. Shoulders square to camera.",
        "focus": "Cut 1: both in plane. Cut 2: @image2 only.",
        "depth": "Shallow DOF"
      },
      "blocking": {
        "location": "Living room - entrance",
        "movement": "Cut 1: @image1 enters and slams door; @image2 bursts in behind him. Cut 2: @image2 solo chest-up.",
        "interaction": "Staggered two-shot, never face-to-face.",
        "positions": [
          { "subjectId": "@image1", "description": "Left third x=34%, foreground, by the door" },
          { "subjectId": "@image2", "description": "Right third x=70%, midground, bursting in" }
        ]
      },
      "acting": {
        "emotion": "Rage",
        "bodyLanguage": "@image1: jaw tight, planted. @image2: chin lifting, jaw driving.",
        "dialogue": "\"Are you seriously gonna do this to me again?!\"",
        "microExpressions": ["Chin jabbing forward", "Hand cutting the air"]
      },
      "timeline": {
        "duration": 11,
        "segments": [
          { "start": 0, "end": 6, "label": "Cut 1 - Staggered two-shot" },
          { "start": 6, "end": 11, "label": "Cut 2 - Single Mike" }
        ],
        "beats": [
          { "start": 0, "end": 2, "description": "@image1 enters, slams door" },
          { "start": 2, "end": 6, "description": "@image2 bursts in, advances" },
          { "start": 6, "end": 11, "description": "@image2 solo, barking" }
        ]
      },
      "audio": {
        "dialogue": "\"Are you seriously gonna do this to me again?!\"",
        "ambient": "Room tone",
        "sfx": ["Door slammed twice", "Footfalls on floorboards"],
        "music": false
      },
      "references": [
        { "slot": "@image1", "assetId": "wyatt", "type": "character" },
        { "slot": "@image2", "assetId": "mike", "type": "character" }
      ],
      "prompt": {
        "en": "Complete Seedance-ready prompt in English. Self-contained visual description, framing, camera movement, acting, audio, capture settings. Max ~3500 chars.",
        "zh": "Complete Seedance-ready prompt in Chinese, same content as 'en'."
      },
      "render": {
        "mode": "M1",
        "engine": "Seedance"
      },
      "notes": {
        "todos": ["Load @image1 Wyatt", "Load @image2 Mike"],
        "warnings": ["Critical warning text"],
        "approved": false
      }
    }
  ]
}

## Field specifications

### Top-level fields
- **description** (string): One-line logline describing the core dramatic conflict (e.g. "Wyatt & Mike - Fraternal conflict in the living room"). Required.
- **duration** (number): Total duration of all shots combined in seconds. **Must exactly match the sum of all shot durations.**
- **mode** (string): Render mode. One of: M1, M2, M3.
- **aspectRatio** (string): Aspect ratio, always "9:16".
- **references** (array of { slot, assetId, type }): Every unique reference used across all shots. slot follows "@imageN" convention. type is one of: "character", "plate", "prop", "environment".
- **sequenceFlow** (object): Time budget and emotional intensity curve. Defines the dramatic arc across time with per-shot intensity values.
- **directorNotes** (object): Director-level metadata — goal, style guide, and sequence-wide warnings.
- **shots** (array): Array of Shot objects, each a self-contained unit with camera, composition, blocking, acting, audio, and prompts.

### sequenceFlow fields
- **title** (string): Budget label (e.g. "Time budget"). Required.
- **subtitle** (string): Short dramatic subtitle of the emotional journey.
- **duration** (number): Hard duration cap in seconds. Total shot durations must NEVER exceed this value.
- **metric** (string): Narrative dimension being tracked. One of: **dramaticIntensity**, **emotion**, **action**, **suspense**, **rhythm**.
- **scale** (object): { start (string), middle (string), end (string) } — descriptive labels for low/mid/high of the intensity curve.
- **segments** (array of objects): One per shot. Each has:
  - **id** (string): Segment identifier, matches shot id.
  - **shotId** (string): Corresponding shot id.
  - **label** (string): Beat label: Hook, Friction, Spike, or Button.
  - **start** (number): Start time in seconds (cumulative from scene start).
  - **end** (number): End time in seconds.
  - **intensity** (number): Dramatic intensity from 0.0 to 1.0.
  - **color** (string): Hex color string (e.g. "#3d8b8f").

### directorNotes fields
- **goal** (string): Single sentence — what should the audience feel or understand from this sequence?
- **styleGuide** (string): Visual and technical constraints packed in one line (e.g. "teal-amber grade - spherical rectilinear lens - flat field no vignette - 24fps 180 degree - diegetic audio only - prompt in positive").
- **warnings** (array of string): Critical sequence-wide warnings. Things NOT to do, character-loading constraints, blocking rules.

### Shot fields (shots[])

- **id** (string): Single capital letter A-Z, sequential. First shot is always "A".
- **title** (string): Dramatic short title capturing the key action or emotion (e.g. "Mike bursts in + explodes").
- **description** (string): One-sentence plain-language action summary. No camera jargon.
- **duration** (number): Shot duration in seconds.
- **start** (number): Cumulative start time in seconds from scene start.
- **end** (number): Cumulative end time in seconds (start + duration).

- **camera** (object, all sub-fields required):
  - **lens** (string): Focal lengths. Use "to" for zoom/push (e.g. "40mm to 55mm"), comma for cut changes (e.g. "55mm, 75mm").
  - **framing** (string): Shot type and composition pattern. EXAMPLE: "Staggered two-shot to single Mike", "Chest-up to Close-up push-in".
  - **movement** (string): Camera motion technique. EXAMPLES: "Handheld", "Push-in lento continuo", "Handheld stable", "Near-static".
  - **fps** (number): Always 24.
  - **shutter** (string): Always "180 degree".
  - **aspectRatio** (string): Always "9:16".

- **composition** (object, all sub-fields required):
  - **frameMap** (string): Frame-by-frame composition. Must include per-cut timing and subject x% positions. EXAMPLE: "Cut 1 (0-6s): Staggered two-shot. @image1 left third x=34%. @image2 right third x=70%. Cut 2 (6-11s): Single @image2."
  - **subjectLock** (string): Visual consistency rules for subjects across cuts. EXAMPLE: "@image1 + @image2: identical face/wardrobe. Shoulders square to camera."
  - **crossFrameRules** (string): Spatial rules across cuts — center line, shoulder orientation, subject relationships. EXAMPLE: "@image1 left, @image2 right - never cross center. Shoulders square to camera."
  - **focus** (string): Focus target per cut.
  - **depth** (string): Depth of field. Always "Shallow DOF".

- **blocking** (object, all sub-fields required):
  - **location** (string): Where in the scene (e.g. "Living room - entrance", "Living room - center").
  - **movement** (string): Per-cut character movement and camera response. EXAMPLE: "Cut 1: @image1 enters and slams door; @image2 bursts in behind him. Cut 2: @image2 solo chest-up."
  - **interaction** (string): Character relationship within the frame (e.g. "Staggered two-shot, never face-to-face.", "Singles only, eyelines connect across cut.").
  - **positions** (array of { subjectId, description }): Exact frame positions. subjectId references the @imageN slot. EXAMPLE: [{ "subjectId": "@image1", "description": "Left third x=34%, foreground, by the door" }].

- **acting** (object, all sub-fields required):
  - **emotion** (string): Core emotion (e.g. "Rage", "Gut-punch / Inward detonation", "Cold demand vs cold fury").
  - **bodyLanguage** (string): Observable physical behavior. Use "@imageN:" prefix per character. EXAMPLE: "@image1: jaw tight, planted. @image2: chin lifting, jaw driving."
  - **dialogue** (string): Key dialogue lines. Empty string "" if silent.
  - **microExpressions** (array of string): Observable micro-beats — facial twitches, hand movements, breath changes. Noun phrases only.

- **timeline** (object, all sub-fields required):
  - **duration** (number): Shot duration in seconds.
  - **segments** (array of { start, end, label }): Sub-segments by cut or action change.
  - **beats** (array of { start, end, description }): Dramatic beats with cumulative timestamps.

- **audio** (object, all sub-fields required):
  - **dialogue** (string): Exact dialogue. Empty string "" if none.
  - **ambient** (string): Environmental background (e.g. "Room tone", "Near-silence", "Low room tone, house creak").
  - **sfx** (array of string): Sound effects as noun phrases (e.g. "Door slammed twice", "Footfalls on floorboards").
  - **music** (boolean): Whether music plays. Default false.

- **references** (array of { slot, assetId, type }): References used in THIS shot. slot is "@imageN", assetId is kebab-case, type is "character" or "plate".
- **prompt** (object):
  - **en** (string): Full self-contained Seedance prompt in English. Max ~3500 chars.
  - **zh** (string): Full Chinese translation of the same prompt.
- **render** (object): { mode (string), engine (string, always "Seedance") } — characterCount is calculated on the frontend.
- **notes** (object):
  - **todos** (array of string): Assets to load or prepare (e.g. "Load @image1 Wyatt").
  - **warnings** (array of string): Shot-specific warnings, overrides or adds to top-level warnings.
  - **approved** (boolean): Whether approved for generation. Default false.

## Critical rules
1. **Total shot durations must sum to sequenceFlow.duration, never exceed it.** Hard cap on total runtime.
2. Every shot's **prompt.en** must be a COMPLETE, SELF-CONTAINED Seedance prompt — do NOT rely on external context. Must include ALL of: scene and mood, frame map with subject x% positions, subject locks, cross-frame rules, character movement and blocking, dialogue and actions, last frame description, camera specs (lens, framing, movement), aspect ratio, fps, shutter, total duration. Write in imperative positive style (what TO do). Max ~3500 chars.
3. Every shot's **prompt.zh** must be the full Chinese translation of the same English prompt — same detail, same completeness. Only include prompt.zh if instructed to do so by the system prompt.
4. **Beats must follow a dramatic arc**: HOOK (opens, sets tone) -> FRICTION (escalation, can have multiple friction shots) -> SPIKE (emotional or action peak) -> BUTTON (resolution, aftermath, closing).
5. **Aim for natural coverage variety**: mix singles, two-shots, OTS, inserts, staggered compositions. Avoid identical framing patterns across consecutive shots.
6. **characterCount (en, zh)** should approximate actual character length of each prompt. Overestimate slightly rather than underestimate.
7. **blocking.positions** must reference subjects using their @imageN slot, never by character name alone.
8. **All timestamps (start, end)** are cumulative from scene start, NOT per-shot relative. First shot always starts at 0.
9. **JSON validity**: All string values must be valid JSON strings. Escape ALL double quotes inside prompts and descriptions using backslash (\\"). Do NOT use unescaped double quotes inside any string field. Use single quotes for dialogue inside prompts.`

const defaultProncerPrompt = `You are a professional cinematography prompt consultant. Your ONLY role is to help refine and optimize video-generation prompts.

Given a current prompt and optional context, the user may ask you to:
- Make the prompt more descriptive or cinematic
- Add specific camera angles, lighting, or atmosphere
- Shorten or restructure the prompt
- Suggest improvements

Return ONLY a valid JSON object:
{
  "optimized_prompt": "the improved prompt",
  "suggestions": ["suggestion 1", "suggestion 2"],
  "changes_made": ["Added camera angle", "Enhanced lighting description"]
}

CRITICAL:
- Do NOT generate shot lists. Do NOT generate scene descriptions.
- Only refine the given prompt.
- Return ONLY valid JSON, no markdown fences, no extra text.`

// ─── Shot Builder ─────────────────────────────────────────────────

func (h *Handler) ClaudeGenerateShots(c *gin.Context) {
	var req ClaudeGenerateShotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Prompt == "" {
		utils.BadRequest(c, "prompt is required")
		return
	}

	// Build final prompt with optional scene context
	finalPrompt := req.Prompt
	if req.SceneContext != nil {
		finalPrompt = buildSceneContextBlock(req.SceneContext) + "\n\n" + req.Prompt
	}

	// Resolve system prompt: skill > request > hardcoded default
	systemPrompt := h.resolveSystemPrompt(req.SkillID, req.SystemPrompt)

	// Append zh-generation instruction when Chinese is disabled
	if !req.GenerateZh {
		systemPrompt += "\n\nIMPORTANT: Do NOT generate Chinese prompts (prompt.zh). Only generate the English prompt (prompt.en) for each shot. Omit the prompt.zh field entirely from the JSON."
	}

	keyModel := req.Model
	if keyModel == "" {
		keyModel = "claude-shot-builder"
	}
	apiModel := req.APIModel
	if apiModel == "" {
		apiModel = keyModel
	}

	reply, err := h.callClaude(c.Request.Context(), keyModel, apiModel, systemPrompt, finalPrompt)
	if err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to generate shots: %v", err))
		return
	}

	utils.Success(c, ClaudeGenerateShotsResponse{
		TaskID: fmt.Sprintf("claude_%d", time.Now().UnixMilli()),
		Model:  apiModel,
		Status: "succeeded",
		Text:   reply,
	})
}

// ─── Proncer ──────────────────────────────────────────────────────

func (h *Handler) ClaudeOptimizePrompt(c *gin.Context) {
	var req ClaudeOptimizePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.CurrentPrompt == "" {
		utils.BadRequest(c, "current_prompt is required")
		return
	}

	// Build the prompt
	var promptParts []string
	promptParts = append(promptParts, fmt.Sprintf("Current prompt:\n%s", req.CurrentPrompt))

	if req.UserInstructions != "" {
		promptParts = append(promptParts, fmt.Sprintf("User instructions:\n%s", req.UserInstructions))
	}

	if req.ShotContext != nil {
		promptParts = append(promptParts, fmt.Sprintf("Shot name: %s", req.ShotContext.ShotName))
		if req.ShotContext.ShotDescription != "" {
			promptParts = append(promptParts, fmt.Sprintf("Shot description: %s", req.ShotContext.ShotDescription))
		}
	}

	if req.SceneContext != nil {
		promptParts = append(promptParts, buildSceneContextBlock(req.SceneContext))
	}

	// Resolve system prompt: skill > request > hardcoded default
	systemPrompt := h.resolveSystemPrompt(req.SkillID, req.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = defaultProncerPrompt
	}

	finalPrompt := strings.Join(promptParts, "\n\n")

	keyModel := req.Model
	if keyModel == "" {
		keyModel = "claude-shot-builder"
	}
	apiModel := req.APIModel
	if apiModel == "" {
		apiModel = keyModel
	}

	reply, err := h.callClaude(c.Request.Context(), keyModel, apiModel, systemPrompt, finalPrompt)
	if err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to optimize prompt: %v", err))
		return
	}

	optimized, suggestions, changes := parseOptimizeResponse(reply)
	utils.Success(c, ClaudeOptimizePromptResponse{
		TaskID:          fmt.Sprintf("claude_%d", time.Now().UnixMilli()),
		Model:           apiModel,
		Status:          "succeeded",
		OptimizedPrompt: optimized,
		Suggestions:     suggestions,
		ChangesMade:     changes,
		RawText:         reply,
	})
}

// ─── Skill resolution ─────────────────────────────────────────────

// resolveSystemPrompt resolves the system prompt to use:
//  1. skill_id → look up skill in DB, use its system_prompt
//  2. request system_prompt (fallback)
//  3. hardcoded default (fallback)
func (h *Handler) resolveSystemPrompt(skillID, requestPrompt string) string {
	if skillID != "" && h.skillSvc != nil {
		skill, err := h.skillSvc.GetByID(skillID)
		if err == nil && skill != nil && skill.SystemPrompt != "" {
			log.Printf("[skill] using skill %q (%s) for system prompt", skill.Name, skill.ID)
			return skill.SystemPrompt
		}
		if err != nil {
			log.Printf("[skill] error looking up skill %q: %v", skillID, err)
		}
	}

	if requestPrompt != "" {
		return requestPrompt
	}

	return defaultShotBuilderPrompt
}

// ─── Core Claude API call ─────────────────────────────────────────

// modelNameMap maps model names (from frontend or DB) to real Anthropic model IDs.
// Add entries here for each model registered in the Providers UI.
// Real Anthropic model IDs are mapped to themselves for passthrough.
var modelNameMap = map[string]string{
	"claude-shot-builder": "claude-sonnet-4-6",
	"claude-assistant":    "claude-sonnet-4-6",
	"claude-sonnet-4-6":   "claude-sonnet-4-6",
	"claude-opus-4-8":     "claude-opus-4-8",
	"claude-haiku-4-5":    "claude-haiku-4-5",
	"claude-fable-5":      "claude-fable-5",
}

func (h *Handler) callClaude(ctx context.Context, keyModel, apiModel, systemPrompt, userPrompt string) (string, error) {
	// 1. Resolve API key from provider store
	apiKey := h.resolveAPIKey(keyModel)

	// 2. Create a per-request client with the resolved API key
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(5*time.Minute),
	)

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     apiModel,
		MaxTokens: 16384,
		System: []anthropic.TextBlockParam{{
			Text: systemPrompt,
			CacheControl: anthropic.CacheControlEphemeralParam{
				TTL: anthropic.CacheControlEphemeralTTLTTL5m,
			},
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		// Try to extract the human-readable message from the API error response.
		msg := err.Error()
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) && apiErr.RawJSON() != "" {
			var body struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if ue := json.Unmarshal([]byte(apiErr.RawJSON()), &body); ue == nil && body.Error.Message != "" {
				msg = body.Error.Message
			}
		}
		return "", fmt.Errorf("Claude API error: %s", msg)
	}

	// Concatenar bloques de texto de la respuesta
	var reply string
	for _, block := range resp.Content {
		if block.Type == "text" {
			reply += block.Text
		}
	}

	log.Printf("[claude] keyModel=%q apiModel=%q tokens: input=%d output=%d cache_read=%d cache_created=%d",
		keyModel, apiModel,
		resp.Usage.InputTokens,
		resp.Usage.OutputTokens,
		resp.Usage.CacheReadInputTokens,
		resp.Usage.CacheCreationInputTokens,
	)

	return reply, nil
}

// resolveAPIKey looks up the API key for a model from the provider store.
func (h *Handler) resolveAPIKey(modelName string) string {
	if h.providerStore == nil {
		return ""
	}

	m, err := h.providerStore.GetModelByName(modelName)
	if err == nil && m != nil && m.APIKey != "" {
		return m.APIKey
	}

	return ""
}

// ─── Helpers ──────────────────────────────────────────────────────

func buildSceneContextBlock(ctx *SceneContext) string {
	var parts []string
	parts = append(parts, "=== Scene Context ===")

	if ctx.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", ctx.Description))
	}

	if len(ctx.Characters) > 0 {
		parts = append(parts, "Characters:")
		for _, c := range ctx.Characters {
			if c.Description != "" {
				parts = append(parts, fmt.Sprintf("  - %s: %s", c.Name, c.Description))
			} else {
				parts = append(parts, fmt.Sprintf("  - %s", c.Name))
			}
		}
	}

	if len(ctx.Presets) > 0 {
		parts = append(parts, "Cinematography presets:")
		for _, p := range ctx.Presets {
			if p.Prompt != "" {
				parts = append(parts, fmt.Sprintf("  - %s (%s): %s", p.Label, p.Code, p.Prompt))
			} else {
				parts = append(parts, fmt.Sprintf("  - %s (%s)", p.Label, p.Code))
			}
		}
	}

	if len(ctx.Assets) > 0 {
		parts = append(parts, "Reference assets:")
		for _, a := range ctx.Assets {
			parts = append(parts, fmt.Sprintf("  - %s (%s)", a.Filename, a.MimeType))
		}
	}

	return strings.Join(parts, "\n")
}

func parseOptimizeResponse(text string) (optimized string, suggestions, changes []string) {
	if text == "" {
		return "", nil, nil
	}

	var parsed struct {
		OptimizedPrompt string   `json:"optimized_prompt"`
		Suggestions     []string `json:"suggestions"`
		ChangesMade     []string `json:"changes_made"`
	}

	cleanText := text
	if idx := strings.Index(text, "{"); idx >= 0 {
		if endIdx := strings.LastIndex(text, "}"); endIdx > idx {
			cleanText = text[idx : endIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(cleanText), &parsed); err == nil {
		return parsed.OptimizedPrompt, parsed.Suggestions, parsed.ChangesMade
	}

	return text, nil, nil
}
