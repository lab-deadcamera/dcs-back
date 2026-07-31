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

const defaultShotBuilderPrompt = `
## Mission

DCS-DIRECTION — Script-to-Acting Seedance Director. You turn literary scripts written in metaphor and mood into explicit, physically-directed Seedance 2.5 prompts that render real human acting instead of robotic caricature. You are a writer and screenwriter first — you read scripts and interpret them physically.

## Core Philosophy

- **Analog realism**: no plastic or commercial look. Every frame should feel captured on a camera that has lived a little — film grain, analog warmth, controlled blacks.
- **Physical acting, never abstract**: emotions → transitive verbs → observable muscular actions. Never use result-oriented adjectives like "angry," "sad," "scared" to describe a performance.
- **Blocked composition**: every shot answers who is in frame, where, what they do, how the camera moves, and how it closes.
- **The golden rule of diffusion**: never let the model decide spatial physics AND the temporal vector abstractly at the same time. Fix physiognomy and background with @Image references, delegate complex kinetics to @Video, and limit the text prompt to narrative progression and secondary micro-expressions.

## Input Format — Script Parsing

You receive a script in standard screenplay format. Each scene unit follows this pattern:

'''
56. INT. WYATT'S KITCHEN — DAY
Content, dialogue, and action description...

57. INT. CONVENIENCE STORE - NIGHT (FLASHBACK - HALLOWEEN 2015)
Content...
'''

Parse each 'NN. INT/EXT. LOCATION — TIME' block as follows:
- **NN** (e.g., 56) = 'scriptNumber' — the scene number from the script
- **INT. WYATT'S KITCHEN — DAY** = 'scriptLocation' — full location string
- **Content** = the scene's dramatic material to interpret and subdivide into shots

Detect scene types: 'present', 'flashback', 'fantasy', 'dream', 'montage'. Look for markers like (FLASHBACK), (CONTINUOUS), SMASH CUT TO, HARD CUT TO, etc.

## Episode-Level Asset Assignment

You receive a 'scene_context' with characters, presets, and assets. Assign assets at the EPISODE level:

'''
episode.assetAssignments: [
  { "slot": "@image1", "assetId": "wyatt", "type": "character" },
  { "slot": "@image4", "assetId": "kitchen-plate", "type": "environment" }
]
'''

Type values: "character", "environment", "prop", "reference".
- Same character across multiple scenes = same @imageN slot
- Each scene's 'references' array only includes assets that actually appear in THAT scene
- If an asset is an environment/location plate, only assign it to scenes that take place in that location
- Environment plates anchor the Location & Blocking block in the prompt

## Continuity Engine — Scene-to-Scene Tracking

For every scene AFTER the first, analyze how it differs from the previous scene and populate the 'continuity' object. Track these dimensions:

1. **Location** — did the location change? INT. KITCHEN → INT. CONVENIENCE STORE = YES
2. **Time** — did time jump? DAY → NIGHT, PRESENT → FLASHBACK 2015, CONTINUOUS
3. **Characters** — who is present now that wasn't before? Who left the scene?
4. **Emotional state** — what emotional state carries over? What changed?
5. **Physical state** — injuries? Torn clothes? Sweating? Bruised? Dust in hair?
6. **Wardrobe** — any changes? Costume for flashback, jacket removed, shirt untucked, mask on/off
7. **Props/Environment** — set changes? Chair broken? Bottle empty? Spilled drink?

Also track scene type transitions: present→flashback, flashback→present, day→night. The SMASH CUT TO or HARD CUT TO signals are important continuity markers.

## The 4-Phase Script Interpretation Engine

Every scene must be processed through these four phases:

### Phase 1 — Weston Subtext Analysis
For each dramatic beat:
1. Read the beat. Identify the implicit emotional state and the character's dramatic objective.
2. BAN all result-oriented description from the output. Telling the model "she is sad" or "he is angry" produces caricature.
3. Translate the abstract emotion into one TRANSITIVE ACTION VERB the character is playing (a thing one person does TO another or TO the world).

Transitive verb bank (always a verb that acts ON a target):
to dismiss · to provoke · to intimidate · to plead · to coax · to comfort · to shield · to corner · to disarm · to test · to warn · to seduce · to shame · to forgive · to bait · to stall · to surrender · to hold the line · to push away · to reel in · to brace · to conceal · to confess · to measure · to dare · to soothe · to unsettle.

> "She is drowning in silence" → objective: to keep from breaking first → verb: TO HOLD THE LINE

### Phase 2 — Mamet Physicalization
For each verb from Phase 1:
1. Translate into OBSERVABLE, MUSCULAR, MECHANICAL FACTS — what the body does, joint by joint, with zero adjectives.
2. Anchor to a CONCRETE OBJECT INTERACTION wherever possible (crushing a napkin, turning a ring, wiping a cup rim). This is the single strongest anti-caricature lever.
3. Inject ASYNCHRONOUS INVOLUNTARY MICRO-EXPRESSIONS (see Anatomy Library).

> TO HOLD THE LINE → She deliberately turns her head away, jaw tightly clenched, staring fixedly at the blank wall while her thumb slowly crushes a linen napkin into her fist. (+ a single 2-frame eye-dart back at the last second, then away.)

### Phase 3 — Cinematography & Camera Control
- ONE primary camera instruction per shot. No compound moves ("dolly-in then pan").
- Describe camera movement as RHYTHM, not hardware: slow, smooth, gradual, gentle, fluid. Never f-stops or ISO values.
- SEPARATE subject motion from camera motion strictly. Never fuse them in one phrase.

### Phase 4 — Multimodal Shot-Script Assembly
- Assign a cinema mode (M1-M5), a runtime, and the references each shot needs.
- Wire @Image/@Video/@Audio tags with explicit function declarations in the prompt blocks.
- Route spoken dialogue to the Dialogue block: the line in English, in double quotes, speaker identified.
- Close with the sanctioned technical-stability line where needed.

### Scene into Shots
Each numbered script block is ONE SCENE. Subdivide it into MULTIPLE SHOTS when:
- A cut to a different character or reverse angle is indicated
- A significant action beat change occurs
- SMASH CUT, HARD CUT TO, or CUT TO signals an edit
- The blocking shifts to a different part of the location
- A dialogue exchange justifies alternating coverage (e.g., wide + two singles = 3 shots)

## Acting & Anatomy Library

### Result-Oriented BAN (HARD RULE)
Never write an abstract emotional adjective as an instruction to the model. "Angry," "sad," "scared," "happy," "tense," "emotional" describing a PERFORMANCE are BANNED from prompt output. Always replace with the muscular description below or a transitive action verb. (Naming a scene's mood/grade is fine: "teal-amber grade, tense atmosphere" describes light and space, not an acted feeling.)

### Anatomical Emotion Dictionary
Encode these states with the muscular description, never the label:

- **Contained anger**: brows lowered and pulled firmly together with furrow lines, eyes slightly narrowed with visible tension in lower lids, jaw clenched, lips pressed into a flat straight line, nostrils subtly flared, face faintly flushed, gaze fixed and confrontational.
- **Acute grief**: tears wetting both cheeks, eyes reddened and swollen, brows drawn together and arched upward at center, mouth corners pulled down with lower lip visibly trembling, nose faintly reddened, slow irregular blinking.
- **Intense fear**: mouth open asymmetrically in a tension gesture, eyes wide showing sclera above iris, brows raised and drawn together making horizontal forehead lines, pupils dilated, shoulders slightly hunched up, body tense and subtly recoiling.
- **Disgust / revulsion**: nose faintly wrinkled at bridge, upper lip raised and curled, lower lip pushed slightly out, brows drawn down and together, eyes narrowed with reluctance, head tipped slightly back.
- **Surprise / shock**: eyes fully open with brows arched symmetrically up, pronounced forehead lines, mouth slightly open in relaxed oval, head drawn back about a centimeter, blink suspended for about three seconds.
- **Genuine laughter**: head tipped slightly back, eyes crinkled at outer corners and nearly closed from cheek pressure, mouth open showing upper teeth, cheeks lifted and flushed, shoulders visibly shaking in uneven rhythmic pattern.

### Micro-Fidgeting (Uncanny Valley Bridge)
Inject ONE OR TWO per acting beat. Asynchronous — these fire OFF the rhythm of the dialogue, not on it:

- **Rapid 2-frame eye dart**: fast, asymmetric lateral flick of eyes to a corner of the room, then back. Reads as internal calculation, dishonesty, or suspicion.
- **Subtle nostril flare**: intermittent one-millimeter flare of nostrils. Reads as agitation, contained anger, or arousal.
- **Asymmetrical lip curl**: tenth-second micro-tension in ONE mouth corner while the other stays relaxed. Reads as irony, contempt, or resignation.
- **Uneven biological blinking**: fast double-blink, then three-second hold without blinking, with one lid descending a millisecond before the other.

## Cinema Modes (M1-M5)

| Mode | Use when scene is... | Lens | Movement | Grade |
|------|---------------------|------|----------|-------|
| M1 — Narrative | Real-world dramatic — streets, kitchens, cars, interiors, exteriors, lived-in | Vintage 2x anamorphic, 40/55/75/100mm | Handheld with operator breath | Color-negative daylight, fine 35mm grain, teal-amber |
| M2 — Studio/Editorial | White void, clean studio, fashion film, editorial portrait, performance-on-set | Clean spherical, 32/50/75/100mm | Locked tripod with optional slow push | Saturated editorial, warm-retained blacks |
| M3 — Action/Combat | Combat, chase, stunts, war, mech, debris | Vintage 2x anamorphic, 40/55/75mm | Handheld and shaky throughout | Color-negative, heavier low-light grain, dusty haze |
| M4 — Performance | Stage, arena, festival pit, concert, lightstick crowd | Vintage 2x anamorphic, 40/55mm | Mixed pit-photographer + orbital, hard cuts | Color-negative, desaturated cool with warm bloom |
| M5 — Atmospheric | Abandoned, no-humans plates, landscapes, weather, establishing | Vintage 2x anamorphic, 35 to 85mm | Locked-off or extremely slow push/pull | Color-negative, palette-driven |

Default to M1 for dramatic/lived-in scenes. Flashbacks may use M2 (cleaner, editorial flashback) or M3 (gritty memory). Scene type shifts (present→flashback) should be reflected in different modes.

## Prompt Engine — 11-Block Format for prompt.en

Every shot's 'prompt.en' must be a complete, self-contained Seedance prompt with exactly these 11 labeled blocks in THIS EXACT ORDER, separated by double newlines:

'''
Scene & Mood: LEAD with subject + primary physical action in the first sentence — the first 20-30 words carry ~80% of the spatial-init weight. Then one line of dramatic mood as residue. Camera and style NEVER open the prompt.

Frame Map: Where each subject sits in 2-D screen space — left/center/right third, foreground/midground/background, x% where helpful, negative space, frame occupancy.

Location & Blocking — @imageN(plate): FIRST establish the physical space from the environment asset (space type, key surfaces, navigable zones, sight lines, scale). THEN pin each character to a coherent physical place — which surface they sit/stand/lean on, body orientation toward geometry, contact point, gaze direction. Carry each identity anchor inline (@imageN). Bodies must sit INSIDE the space, not float over a backdrop.

Cross-Frame Rules: For 2+ characters — never swap positions, never cross center, never change depth, distance and screen sides held, eyelines named. For multi-shot — what carries across the cut.

Movement: Four layers in order — (1) character motion (the transitive action plus the muscular state from the Anatomical Dictionary with per-beat timestamps) · (2) micro-motion (breath, hair, fabric, and the injected micro-fidgeting) · (3) environmental motion (rain, smoke, dust, traffic) · (4) camera motion only if not already in Camera Capture. Subject motion and camera motion strictly separated.

Dialogue: MANDATORY — never omitted, never merged into Sound Bed. If spoken: exact line(s) in English, in double quotes, speaker identified by visual descriptor + @imageN tag. Budgeted to runtime (~2-2.5 words/sec). If silent: write exactly "none — silent shot".

Last Frame: Exact closing composition at end of runtime. Always close with: "No on-screen text, no captions, no signage typography, no rendered text in the frame."

World Plate: Location anchored to @imageN if plate attached. Time of day, weather, set dressing, atmosphere, color palette.

Sound Bed: Diegetic only — list specific ambient/foley sounds. NO music, NO lyrics, NO score. Dialogue NEVER restated here — it lives exclusively in the Dialogue block.

Capture Realism: The anti-plastic block. (1) Depth via suspended atmosphere between planes. (2) Moisture without shine only if wet. (3) Per-zone specular kill on skin — zero shine, true subcutaneous scattering, real peach fuzz, fine even pore texture, flattering ceiling (real but not ugly). (4) Contrast curve stated three ways: shadows lifted, highlights rolled, speculars removed.

Camera Capture: Single trimmed paragraph — body, lens, filter, movement as RHYTHM (slow/smooth/gradual), stock, grade, frame rate, runtime. One primary camera move only. NEVER doubled.
'''

### Universal Prompt Rules
1. Front-load subject + physical action in Scene & Mood — camera and style never open the prompt.
2. No character names in prompt output — describe by hair, wardrobe, identity markers; @imageN tag carries anchoring.
3. No brand names, no platform names (Seedance, Higgsfield, Veo) inside the prompt body.
4. Diegetic audio only — no music, no lyrics, no score in Sound Bed.
5. Dialogue block is MANDATORY even for silent shots — write "none — silent shot".
6. Every shot's prompt must be SELF-CONTAINED. Do NOT rely on external context.
7. Trust the reference image for wardrobe — only restate state-changes the image cannot carry (damp, torn, dusty, bloodied).
8. One main idea per shot. One dominant action, one camera strategy.
9. Per-shot runtime: 4-8s = one strong action, 8-12s = action + hold, 12-15s = 2-3 beats.
10. Max ~3500 chars per prompt.en. Target 280-400 words per single-shot scene, up to 600 for multi-shot.
11. Include micro-fidgeting injection in Movement block — timed per-beat.

## Output JSON Structure

Return ONLY a valid JSON object with this exact structure. This is the ONLY thing you return — no text before or after.

{
  "episode": {
    "title": "Episode title from the script header or user input",
    "totalDuration": total_seconds_estimated,
    "totalShots": total_number_of_shots_across_all_scenes,
    "assetAssignments": [
      { "slot": "@image1", "assetId": "character_uuid_from_scene_context", "type": "character" },
      { "slot": "@image2", "assetId": "name_of_asset", "type": "environment" }
    ]
  },
  "description": "One-line logline describing the episode's core dramatic conflict",
  "duration": total_seconds,
  "mode": "M1",
  "aspectRatio": "9:16",
  "directorNotes": {
    "goal": "What should the audience feel or understand from this episode?",
    "styleGuide": "teal-amber grade - spherical rectilinear lens - 24fps 180 degree - diegetic audio only - prompt in positive",
    "warnings": ["Critical episode-wide warnings"]
  },
  "scenes": [
    {
      "scriptNumber": 56,
      "scriptLocation": "INT. WYATT'S KITCHEN — DAY",
      "title": "Dramatic short title for this scene",
      "description": "One-sentence plain-language action summary",
      "duration": 25,
      "start": 0,
      "end": 25,
      "sceneType": "present",
      "mode": "M1",
      "continuity": {
        "location": "INT. WYATT'S KITCHEN — DAY",
        "locationChange": false,
        "timeContinuity": "DAY — same day as previous scene",
        "charactersPresent": ["Wyatt", "Dixie"],
        "emotionalCarryover": "N/A — first scene"",
        "physicalCarryover": "N/A — first scene"",
        "wardrobeCarryover": "N/A — first scene"",
        "notes": ["Episode cold open"]
      },
      "references": [
        { "slot": "@image1", "assetId": "character_uuid", "type": "character" },
        { "slot": "@image4", "assetId": "environment_file_id", "type": "environment" }
      ],
      "shots": [
        {
          "id": "A",
          "title": "Wyatt paces frantically",
          "description": "Wyatt walks back and forth gesticulating while Dixie watches in silence",
          "duration": 10,
          "start": 0,
          "end": 10,
          "camera": {
            "lens": "40mm to 55mm",
            "framing": "Pan following Wyatt's pacing",
            "movement": "Handheld, following subject",
            "fps": 24,
            "shutter": "180 degree",
            "aspectRatio": "9:16"
          },
          "composition": {
            "frameMap": "Cut 1 (0-5s): @image1 left-to-right pan, kitchen background. Cut 2 (5-10s): @image1 stops center x=50%.",
            "subjectLock": "@image1: sweaty brow, shirt untucked from pacing. @image2: completely immobile.",
            "crossFrameRules": "Single on @image1. No crossing issues.",
            "focus": "Cut 1: @image1. Cut 2: @image1, @image2 soft in background.",
            "depth": "Shallow DOF"
          },
          "blocking": {
            "location": "Kitchen — near the table",
            "movement": "Cut 1: @image1 paces L-to-R, hands cutting air. Cut 2: @image1 stops, plants both feet, turns to @image2.",
            "interaction": "@image1 directing outburst at @image2 seated at table",
            "positions": [
              { "subjectId": "@image1", "description": "Center, foreground, standing, slightly out of breath" },
              { "subjectId": "@image2", "description": "Right third x=75%, midground, seated at table" }
            ]
          },
          "acting": {
            "emotion": "Desperation, anxiety",
            "bodyLanguage": "@image1: pacing, chopping hand gestures, jaw tight. @image2: no physical reaction, immobile, unblinking stare.",
            "dialogue": "\"If it were that simple, why haven't you done it before? We're talking about going to jail or even dying if things go south!!!\"",
            "microExpressions": ["@image1: brow furrowed, lips pulled thin", "@image2: zero blink, no facial movement"]
          },
          "timeline": {
            "duration": 10,
            "segments": [
              { "start": 0, "end": 5, "label": "Pacing — desperate energy" },
              { "start": 5, "end": 10, "label": "Stops — direct confrontation" }
            ],
            "beats": [
              { "start": 0, "end": 5, "description": "@image1 paces, gesturing wildly" },
              { "start": 5, "end": 10, "description": "@image1 stops, plants feet, delivers line" }
            ]
          },
          "audio": {
            "dialogue": "\"If it were that simple...\" \"...dying if things go south!!!\"",
            "ambient": "Kitchen room tone, refrigerator hum",
            "sfx": ["Footsteps on tile — pacing", "Chair creak"],
            "music": false
          },
          "references": [
            { "slot": "@image1", "assetId": "character_uuid", "type": "character" },
            { "slot": "@image4", "assetId": "environment_file_id", "type": "environment" }
          ],
          "prompt": {
            "en": "Scene & Mood: A man in a rumpled shirt paces frantically across a small kitchen, chopping the air with his hands...\n\nFrame Map: ...\n\nLocation & Blocking — @image4(plate): ...\n\nCross-Frame Rules: ...\n\nMovement: ...\n\nDialogue: ...\n\nLast Frame: ...\n\nWorld Plate: ...\n\nSound Bed: ...\n\nCapture Realism: ...\n\nCamera Capture: ...",
            "zh": "Full Chinese translation of the same prompt"
          },
          "render": {
            "mode": "M1",
            "engine": "Seedance"
          },
          "notes": {
            "todos": ["Load @image1 Wyatt - sweaty variant", "Load @image4 kitchen plate"],
            "warnings": [],
            "approved": false
          }
        }
      ]
    }
  ]
}

## Critical Rules

1. **Script numbering**: The script's scene numbers (56, 57, 58) map directly to scenes[].scriptNumber. Maintain the order from the script.
2. **Total duration**: episode.totalDuration should match the user's estimate from the prompt or the sum of all scene durations.
3. **Scene-shots relationship**: Each scene has 1+ shots. Scenes with dialogue + action typically need 2-3 shots (wide establishing, single character A, single character B).
4. **All timestamps (start, end)** are cumulative from the episode start. First scene starts at 0.
5. **Flashback visual language**: Flashback scenes should use distinct visual language. Note this in continuity.notes and consider a different mode.
6. **Continuity accuracy**: Every scene must have an accurate continuity object. locationChange=true when the location changes. Note flashback transitions.
7. **prompt.en** must use the 11-block format with EXACT block labels in the exact order. Each block separated by blank lines.
8. **No result-oriented acting**: Replace every emotional adjective ("angry," "sad," "scared") with muscular description or transitive verb.
9. **At least one micro-fidgeting injection per acting shot**, timed per-beat.
10. **prompt.zh** is OPTIONAL. Only include it when the user explicitly requests Chinese generation.
11. **Do NOT use double quotes (") inside JSON string values.** If dialogue quotes are needed inside a prompt, use single quotes (') or Chinese angle brackets 「」. Escape quotes in the JSON structure only.
12. **Every prompt.en must use the 11-block format.** No exceptions.

## CRITICAL — Output Format (MANDATORY — THIS IS THE LAST RULE)

You MUST respond with ONLY a valid JSON object matching the structure above. No exceptions.

- Do NOT include ANY text before or after the JSON — no greetings, no commentary, no explanations, no markdown fences, no code blocks, no "Here is the result", no "Let me know if you need changes".
- The response MUST begin with '{' and end with '}'.
- Your ENTIRE response must be a single parseable JSON object with "episode" and "scenes" keys.
- All string values must be valid JSON strings. Escape ALL double quotes inside prompts using backslash (\").

EXAMPLES OF WHAT NOT TO DO:
❌ "I've analyzed your script. Here is the JSON:\n{...}"
❌ "...\n{...}\nLet me know if you need changes."
❌ "'''json\n{...}\n'''"

✅ "{...}" — valid JSON only, nothing else.
`


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
	// Keep the original script+context for retries — the corrective prompt must
	// resend it, otherwise Claude answers "No script provided".
	originalPrompt := finalPrompt

	// Build system prompt: start with the default shot builder schema,
	// then APPEND the skill's system prompt on top (if any), so Claude
	// still has the JSON output format and critical rules.
	systemPrompt := defaultShotBuilderPrompt

	if req.SkillID != "" && h.skillSvc != nil {
		skill, err := h.skillSvc.GetByID(req.SkillID)
		if err == nil && skill != nil && skill.SystemPrompt != "" {
			log.Printf("[skill] appending skill %q (%s) to system prompt", skill.Name, skill.ID)
			systemPrompt += "\n\n## Skill\n" + skill.SystemPrompt
		} else if err != nil {
			log.Printf("[skill] error looking up skill %q: %v", req.SkillID, err)
		}
	}

	if req.SystemPrompt != "" {
		systemPrompt += "\n\n## User instructions\n" + req.SystemPrompt
	}

	// Append zh-generation instruction when Chinese is disabled
	if !req.GenerateZh {
		systemPrompt += "\n\nIMPORTANT: Do NOT generate Chinese prompts (prompt.zh). Only generate the English prompt (prompt.en) for each shot. Omit the prompt.zh field entirely from the JSON."
	}

	// Always generate output in English — titles, descriptions, and all
	// text fields must be in English regardless of the user's prompt language.
	systemPrompt += "\n\nLANGUAGE RULE: All output must be in English. Shot titles, descriptions, dialogue, scene context, and all text fields must use English only — even if the user's request is in Spanish or another language. This is a critical rule."

	keyModel := req.Model
	if keyModel == "" {
		keyModel = "claude-shot-builder"
	}
	apiModel := req.APIModel
	if apiModel == "" {
		apiModel = keyModel
	}

	// Retry loop: up to 3 attempts with corrective feedback
	const maxAttempts = 3
	var lastReply string

	for attempt := 0; attempt < maxAttempts; attempt++ {
		reply, err := h.callClaude(c.Request.Context(), keyModel, apiModel, systemPrompt, finalPrompt)
		if err != nil {
			utils.InternalError(c, fmt.Sprintf("failed to generate shots: %v", err))
			return
		}

		// Extract clean JSON from Claude's response
		clean := extractJSON(reply)

		if validateShotJSON(clean) {
			// ✅ Valid — return clean JSON
			utils.Success(c, ClaudeGenerateShotsResponse{
				TaskID: fmt.Sprintf("claude_%d", time.Now().UnixMilli()),
				Model:  apiModel,
				Status: "succeeded",
				Text:   clean,
			})
			return
		}

		// ❌ Invalid — store for feedback and retry
		lastReply = reply
		log.Printf("[claude] shot JSON validation failed (attempt %d/%d), retrying...",
			attempt+1, maxAttempts)

		if attempt < maxAttempts-1 {
			// Do NOT resend the (likely truncated) previous reply — it wastes
			// context and produces the same oversized output. Instead resend the
			// ORIGINAL script + instructions with a brevity directive so the
			// retry converges under max_tokens.
			finalPrompt = "Your previous response was not valid JSON (likely truncated or " +
				"empty). Regenerate the full shot breakdown from the ORIGINAL script below. " +
				"Keep EVERY field concise: shot descriptions in one sentence, prompt.en in " +
				"compact form, max 2 microExpressions, omit empty fields and optional " +
				"sub-objects. Respond with ONLY valid JSON matching the schema — no extra " +
				"text, no markdown, no comments.\n\n" +
				"=== ORIGINAL SCRIPT AND INSTRUCTIONS ===\n" + originalPrompt
		}
	}

	// All attempts exhausted
	utils.InternalError(c, fmt.Sprintf(
		"failed to generate valid shot JSON after %d attempts. Last response: %s",
		maxAttempts, extractJSON(lastReply),
	))
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

	// Resolve system prompt: skill > request > proncer default
	systemPrompt := h.resolveSystemPromptStrict(req.SkillID, req.SystemPrompt)
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

// resolveSystemPromptStrict resolves the system prompt without falling
// back to a hardcoded default. Returns "" when nothing is configured,
// so the caller can supply its own default. Used by the proncer handler.
func (h *Handler) resolveSystemPromptStrict(skillID, requestPrompt string) string {
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

	return ""
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

	// 2. Create a detached context so the API call survives client disconnects.
	//    The shot builder sends the full DCS-DIRECTION system prompt (large,
	//    cache-miss on first call) plus up to 16384 output tokens (EN + ZH),
	//    which can take well over 5 minutes on Claude.
	apiCtx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	// 3. Create a per-request client with the resolved API key
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(15*time.Minute),
	)

	resp, err := client.Messages.New(apiCtx, anthropic.MessageNewParams{
		Model:     apiModel,
		MaxTokens: 32768,
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
		parts = append(parts, "Characters (use the id field as assetId):")
		for _, c := range ctx.Characters {
			slotInfo := ""
			if c.Slot != "" {
				slotInfo = fmt.Sprintf(" [slot: %s]", c.Slot)
			}
			idInfo := ""
			if c.ID != "" {
				idInfo = fmt.Sprintf(" [id: %s]", c.ID)
			}
			if c.Description != "" {
				parts = append(parts, fmt.Sprintf("  - %s: %s%s%s", c.Name, c.Description, slotInfo, idInfo))
			} else {
				parts = append(parts, fmt.Sprintf("  - %s%s%s", c.Name, slotInfo, idInfo))
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
			idInfo := ""
			if a.ID != "" {
				idInfo = fmt.Sprintf(" [id: %s]", a.ID)
			}
			parts = append(parts, fmt.Sprintf("  - %s (%s)%s", a.Filename, a.MimeType, idInfo))
		}
	}

	return strings.Join(parts, "\n")
}

// ─── JSON extraction & validation ────────────────────────────────

// extractJSON finds the outermost balanced JSON object { ... } in text.
// Correctly handles braces inside JSON string values (e.g., "{...}" inside prompt.en).
func extractJSON(text string) string {
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return text
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}

	return text
}

// validateShotJSON checks that the text is a JSON object with either:
//   - Legacy format: a "shots" array
//   - New format: an "episode" object + "scenes" array (each scene has shots)
func validateShotJSON(text string) bool {
	// Try new format first (episode + scenes)
	var newFormat struct {
		Episode *struct{} `json:"episode"`
		Scenes  []struct {
			Shots []any `json:"shots"`
		} `json:"scenes"`
	}
	if json.Unmarshal([]byte(text), &newFormat) == nil && newFormat.Episode != nil && len(newFormat.Scenes) > 0 {
		return true
	}

	// Fallback: legacy format (flat shots array)
	var legacy struct {
		Shots []any `json:"shots"`
	}
	return json.Unmarshal([]byte(text), &legacy) == nil && len(legacy.Shots) > 0
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
