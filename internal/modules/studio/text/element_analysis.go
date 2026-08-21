package text

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

// ─── Element Elicitation (analyze-elements) ──────────────────────
//
// The elicitation pass runs BEFORE generate-shots: Claude reads the decoded
// script and returns a structured element_registry of every visual entity,
// flagging which ones are underspecified ("mentioned but never described").
// The user resolves each gap in the elicitation UI; the resolved registry is
// then sent along with generate-shots / refine-shots, where buildClosedWorldBlock
// turns it into hard visual rules injected into the system prompt.

// validElementCategories is the closed set of categories the analysis prompt
// may emit. Must stay in sync with elementElicitationPrompt.
var validElementCategories = map[string]bool{
	"character":     true,
	"animal":        true,
	"prop":          true,
	"location":      true,
	"vehicle":       true,
	"weather":       true,
	"wardrobe":      true,
	"screen_content": true,
	"sound_object":  true,
	"other":         true,
}

// validAnalysisStatuses is the closed set of definition_status values the
// analysis may emit ("pending" and the user-decision statuses only exist in
// the UI / generate-refine payloads, never in raw analysis output).
var validAnalysisStatuses = map[string]bool{
	"defined":      true,
	"asset_orphan": true,
	"undefined":    true,
}

// elementRegistryPayload mirrors the top-level JSON object the analysis must
// return (see the CRITICAL — Output Format section of elementElicitationPrompt).
type elementRegistryPayload struct {
	ElementRegistry []ElementEntity `json:"element_registry"`
	Summary         string          `json:"summary"`
}

// validateElementRegistryJSON validates a clean analyze-elements response.
// It enforces the non-empty registry plus the required fields and closed
// enums on every entry, so a malformed payload never reaches the UI.
func validateElementRegistryJSON(clean string) error {
	var payload elementRegistryPayload
	if err := json.Unmarshal([]byte(clean), &payload); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	if len(payload.ElementRegistry) == 0 {
		return fmt.Errorf("element_registry must be a non-empty array")
	}
	for i, e := range payload.ElementRegistry {
		if e.EntityID == "" {
			return fmt.Errorf("element_registry[%d].entity_id is required", i)
		}
		if e.ConsistencyGroup == "" {
			return fmt.Errorf("element_registry[%d].consistency_group is required", i)
		}
		if !validElementCategories[e.Category] {
			return fmt.Errorf("element_registry[%d].category %q is not a valid category", i, e.Category)
		}
		if !validAnalysisStatuses[e.DefinitionStatus] {
			return fmt.Errorf("element_registry[%d].definition_status %q is not a valid status", i, e.DefinitionStatus)
		}
		if e.SceneNumber < 0 {
			return fmt.Errorf("element_registry[%d].scene_number must be >= 0", i)
		}
	}
	return nil
}

// runClaudeElementAnalysis is the sibling runner of runClaudeShotBuilder for
// the analyze-elements call: up to 3 attempts with corrective feedback, JSON
// extraction + validation, and log persistence on both success and failure
// (mode "analyze"). It sends no reference images and fires no push — the
// client polls the task endpoint for the registry. On success it returns the
// clean JSON and an empty message; on failure it returns the error message.
func (h *Handler) runClaudeElementAnalysis(
	ctx context.Context,
	user shotBuilderUser,
	meta *shotBuilderMeta,
	systemPrompt, originalPrompt string,
	rawBody []byte,
	skillName, keyModel, apiModel string,
) (string, string) {
	// Retry loop: up to 3 attempts with corrective feedback
	const maxAttempts = 3
	var lastReply string

	start := time.Now()
	var attempts []*ShotBuilderAttempt
	totalInputTokens, totalOutputTokens := 0, 0
	finalPrompt := originalPrompt

	for attempt := 0; attempt < maxAttempts; attempt++ {
		reply, usage, duration, callErr := h.callClaude(ctx, keyModel, apiModel, systemPrompt, finalPrompt, nil)

		a := &ShotBuilderAttempt{
			AttemptNumber: attempt + 1,
			Prompt:        finalPrompt,
			DurationMs:    duration.Milliseconds(),
		}

		if callErr != nil {
			a.ErrorMessage = callErr.Error()
			attempts = append(attempts, a)
			msg := fmt.Sprintf("failed to analyze elements: %v", callErr)
			h.persistLog("failed", msg, user, meta, systemPrompt, originalPrompt, rawBody, skillName, keyModel, apiModel, attempts, totalInputTokens, totalOutputTokens, lastReply, start)
			return "", msg
		}

		clean := extractJSON(reply)
		a.Response = reply
		a.Valid = validateElementRegistryJSON(clean) == nil
		if usage != nil {
			a.InputTokens = int(usage.InputTokens)
			a.OutputTokens = int(usage.OutputTokens)
			a.CacheReadTokens = int(usage.CacheReadInputTokens)
			a.CacheCreationTokens = int(usage.CacheCreationInputTokens)
			totalInputTokens += a.InputTokens
			totalOutputTokens += a.OutputTokens
		}
		attempts = append(attempts, a)

		if a.Valid {
			h.persistLog("succeeded", "", user, meta, systemPrompt, originalPrompt, rawBody, skillName, keyModel, apiModel, attempts, totalInputTokens, totalOutputTokens, clean, start)
			return clean, ""
		}

		lastReply = reply
		log.Printf("[claude] element registry validation failed (attempt %d/%d), retrying...",
			attempt+1, maxAttempts)

		if attempt < maxAttempts-1 {
			// Resend the ORIGINAL input (never the previous reply) with an
			// element-specific corrective directive.
			finalPrompt = "Your previous response was not a valid element-registry JSON object. " +
				"Regenerate the FULL element_registry from the original input below. Respond with ONLY " +
				"a valid JSON object matching the schema — no extra text, no markdown, no comments.\n\n" +
				"=== ORIGINAL SCRIPT AND INSTRUCTIONS ===\n" + originalPrompt
		}
	}

	msg := fmt.Sprintf(
		"failed to generate valid element-registry JSON after %d attempts. Last response: %s",
		maxAttempts, extractJSON(lastReply),
	)
	h.persistLog("failed", msg, user, meta, systemPrompt, originalPrompt, rawBody, skillName, keyModel, apiModel, attempts, totalInputTokens, totalOutputTokens, lastReply, start)
	return "", msg
}

// HandleClaudeAnalyzeElements starts a background element-analysis task over
// the decoded script. The response reuses the async shape of generate-shots:
// {taskId, model, status:"processing"}; the client polls the status endpoint.
func (h *Handler) HandleClaudeAnalyzeElements(c *gin.Context) {
	// Capture the raw request body BEFORE binding — same ground-truth pattern
	// as claudeGenerateShots for reconstructing failed requests in logs.
	rawBody, err := c.GetRawData()
	if err != nil {
		utils.BadRequest(c, fmt.Sprintf("failed to read request body: %v", err))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

	var req ClaudeAnalyzeElementsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Build final prompt with optional scene context (characters + assets the
	// model can link against when flagging asset_orphan entries).
	finalPrompt := req.Prompt
	if req.SceneContext != nil {
		finalPrompt = buildSceneContextBlock(req.SceneContext) + "\n\n" + req.Prompt
	}
	originalPrompt := finalPrompt

	// System prompt: skill override > explicit request prompt > built-in
	// elicitation prompt. Never falls back to the shot-builder default.
	systemPrompt := h.resolveSystemPromptStrict(req.SkillID, req.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = elementElicitationPrompt
	}

	keyModel := req.Model
	if keyModel == "" {
		keyModel = "claude-shot-builder"
	}
	apiModel := req.APIModel
	if apiModel == "" {
		apiModel = keyModel
	}

	meta := &shotBuilderMeta{
		Mode:      "analyze",
		ProjectID: req.ProjectID,
		SceneID:   req.SceneID,
		SkillID:   req.SkillID,
		UserID:    req.UserID,
		UserName:  req.UserName,
	}

	taskID := fmt.Sprintf("claude_elements_%d", time.Now().UnixMilli())
	user := userFromContext(c)

	task := &ShotTask{
		TaskID:    taskID,
		Status:    ShotTaskProcessing,
		Model:     apiModel,
		CreatedAt: time.Now(),
	}
	h.taskStore.Set(task)

	go func() {
		ctx := context.Background()
		clean, errMsg := h.runClaudeElementAnalysis(ctx, user, meta, systemPrompt, originalPrompt, rawBody, "", keyModel, apiModel)
		if errMsg != "" {
			h.taskStore.Update(taskID, func(t *ShotTask) {
				t.Status = ShotTaskFailed
				t.Error = errMsg
			})
			return
		}
		h.taskStore.Update(taskID, func(t *ShotTask) {
			t.Status = ShotTaskSucceeded
			t.Text = clean
		})
	}()

	utils.Success(c, ClaudeGenerateShotsResponse{
		TaskID: taskID,
		Model:  apiModel,
		Status: "processing",
	})
}

// HandleClaudeAnalyzeElementsStatus returns the current state of a background
// element-analysis task. Same polling contract as GetClaudeShotsStatus.
func (h *Handler) HandleClaudeAnalyzeElementsStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "taskId is required")
		return
	}

	task, ok := h.taskStore.Get(taskID)
	if !ok {
		utils.NotFound(c, "element analysis task not found")
		return
	}

	utils.Success(c, ClaudeShotsStatusResponse{
		TaskID: task.TaskID,
		Model:  task.Model,
		Status: task.Status,
		Text:   task.Text,
		Error:  task.Error,
	})
}

// ─── Closed-world injection (generate-shots / refine-shots) ──────

// buildClosedWorldBlock renders a resolved element registry as a hard-rules
// block appended to the shot builder system prompt. Entries without a user
// decision keep their analysis status as guidance; decided entries become
// binding directives keyed by decision type.
func buildClosedWorldBlock(registry []ElementEntity) string {
	var b strings.Builder
	b.WriteString("\n\n## Closed-World Mode (element registry)\n\n")
	b.WriteString("The following visual elements were extracted from this script and RESOLVED by the user. These are HARD rules that override any inference:\n\n")

	for _, e := range registry {
		line := fmt.Sprintf("- [%s] \"%s\" (scene %d)", e.Category, e.MentionedAs, e.SceneNumber)
		switch {
		case e.UserDecision != nil && e.UserDecision.Type == "define_with_reference":
			line += ": USE the linked asset reference EXACTLY as provided"
			if e.UserDecision.Description != "" {
				line += " — " + e.UserDecision.Description
			}
		case e.UserDecision != nil && e.UserDecision.Type == "define_with_text":
			line += ": render EXACTLY as described — " + e.UserDecision.Description
		case e.UserDecision != nil && e.UserDecision.Type == "invent_free":
			line += ": you may invent freely, but keep it consistent across all shots"
		case e.UserDecision != nil && e.UserDecision.Type == "invent_restricted":
			line += ": invent WITHIN these constraints — " + e.UserDecision.Description
		case e.UserDecision != nil && e.UserDecision.Type == "abstract":
			line += ": render as abstract/atmospheric presence only — no concrete form"
		default:
			line += ": status " + e.DefinitionStatus
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\nDo NOT introduce new named visual elements beyond this registry. Do NOT contradict any rule above.\n")
	return b.String()
}
