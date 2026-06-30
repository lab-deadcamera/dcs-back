package text

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"dcs-back-v0/internal/modules/provider"
	"dcs-back-v0/internal/utils"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	providerStore *provider.Store
}

func NewHandler(providerStore *provider.Store) *Handler {
	return &Handler{providerStore: providerStore}
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

	// Default system prompt for shot builder

	systemPrompt := `Eres un director de fotografía y narrative designer. A partir de una descripción de escena, archivos de referencia e instrucciones del usuario, genera un shotlist estructurado para un spot publicitario cinematográfico.

Devuelve ÚNICAMENTE un objeto JSON válido con esta estructura exacta — sin markdown, sin comentarios, sin texto adicional:

{
  "title": "Título del proyecto",
  "scene": "Ubicación — Escena — Continuidad",
  "totalDuration": 80,
  "durationCap": 80,
  "shots": [
    {
      "id": "A",
      "beat": "HOOK",
      "duration": 11,
      "cuts": 2,
      "title": "Nombre del plano",
      "spike": false,
      "prompt": "Prompt completo EN para Seedance (descripción visual, encuadre, movimiento, actuación, sonido, captura)",
      "promptZh": "Prompt completo 中文 para Seedance",
      "guide": {
        "scene": "Descripción de la escena",
        "type": "Tipo de plano",
        "cuts": [["1", "0-6s descripción"], ["2", "6-11s descripción"]],
        "important": "Nota clave de dirección"
      }
    }
  ],
  "conventions": [
    {"label": "Formato", "value": "9:16 vertical"},
    {"label": "FPS", "value": "24fps - 180°"}
  ],
  "faceToFaceRule": "Regla cara-a-cara para planos con más de un personaje"
}

CRITICAL: La duración total de todos los shots NO debe exceder durationCap.
CRITICAL: shots[].prompt debe ser un prompt completo y auto-contenido para generación de video Seedance en inglés.
CRITICAL: shots[].promptZh debe ser la traducción al chino del mismo prompt.`

	modelName := req.Model
	if modelName == "" {
		modelName = "claude-shot-builder"
	}

	reply, err := h.callClaude(c.Request.Context(), modelName, systemPrompt, finalPrompt)
	if err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to generate shots: %v", err))
		return
	}

	utils.Success(c, ClaudeGenerateShotsResponse{
		TaskID: fmt.Sprintf("claude_%d", time.Now().UnixMilli()),
		Model:  modelName,
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

	// Default system prompt for proncer
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = `You are a professional cinematography prompt consultant. Your ONLY role is to help refine and optimize video-generation prompts.

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
	}

	finalPrompt := strings.Join(promptParts, "\n\n")

	modelName := req.Model
	if modelName == "" {
		modelName = "claude-shot-builder"
	}

	reply, err := h.callClaude(c.Request.Context(), modelName, systemPrompt, finalPrompt)
	if err != nil {
		utils.InternalError(c, fmt.Sprintf("failed to optimize prompt: %v", err))
		return
	}

	optimized, suggestions, changes := parseOptimizeResponse(reply)
	utils.Success(c, ClaudeOptimizePromptResponse{
		TaskID:          fmt.Sprintf("claude_%d", time.Now().UnixMilli()),
		Model:           modelName,
		Status:          "succeeded",
		OptimizedPrompt: optimized,
		Suggestions:     suggestions,
		ChangesMade:     changes,
		RawText:         reply,
	})
}

// ─── Core Claude API call ─────────────────────────────────────────

// Maps internal model names to real Anthropic model IDs.
// Add entries here for each model registered in the Providers UI.
var modelNameMap = map[string]string{
	"claude-shot-builder": "claude-sonnet-4-6",
	"claude-assistant":    "claude-sonnet-4-6",
}

func (h *Handler) callClaude(ctx context.Context, internalModel, systemPrompt, userPrompt string) (string, error) {
	// 1. Resolve API key from provider store
	apiKey := h.resolveAPIKey(internalModel)

	// 2. Resolve real Anthropic model ID
	apiModel := modelNameMap[internalModel]
	if apiModel == "" {
		apiModel = internalModel // passthrough — permite cualquier modelo real
	}

	// 3. Create a per-request client with the resolved API key
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(3*time.Minute),
	)

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     apiModel,
		MaxTokens: 8192,
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
		return "", fmt.Errorf("Claude API error: %w", err)
	}

	// Concatenar bloques de texto de la respuesta
	var reply string
	for _, block := range resp.Content {
		if block.Type == "text" {
			reply += block.Text
		}
	}

	log.Printf("[claude] model=%q apiModel=%q tokens: input=%d output=%d cache_read=%d cache_created=%d",
		internalModel, apiModel,
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
