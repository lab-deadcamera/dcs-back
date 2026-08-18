package text

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// PushNotifier is satisfied by the push module's service. Injected via the
// constructor so the shot builder can alert the requesting user when a
// breakdown finishes generating.
type PushNotifier interface {
	SendToUser(userID int64, title, body string, data map[string]string)
}

// VisionImageProvider resolves a stored file id to a publicly reachable URL of
// its vision-size rendering. Satisfied by the file module's Service. The
// backend builds the URL from the id (never from client-supplied URLs) so the
// shot builder only sends images it actually owns, and only image files.
type VisionImageProvider interface {
	VisionURL(id string) (string, error)
}

// visionImage is a single reference image sent to Claude as a vision block,
// with a short label so the model can associate it with a slot / role.
type visionImage struct {
	URL   string
	Label string
}

type Handler struct {
	providerStore  *provider.Store
	skillSvc       *skillmodule.Service
	logStore       *LogStore
	pushSvc        PushNotifier
	vision         VisionImageProvider
	maxOutputTokens int
	maxVisionImages int
}

func NewHandler(
	providerStore *provider.Store,
	skillSvc *skillmodule.Service,
	logStore *LogStore,
	pushSvc PushNotifier,
	vision VisionImageProvider,
	maxOutputTokens, maxVisionImages int,
) *Handler {
	return &Handler{
		providerStore:   providerStore,
		skillSvc:        skillSvc,
		logStore:        logStore,
		pushSvc:         pushSvc,
		vision:          vision,
		maxOutputTokens: maxOutputTokens,
		maxVisionImages: maxVisionImages,
	}
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
- **The golden rule of diffusion**: never let the model decide spatial physics AND the temporal vector abstractly at the same time. Fix physiognomy and background with [Image] references, delegate complex kinetics to [Video], and limit the text prompt to narrative progression and secondary micro-expressions.

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
  { "slot": "[Image1]", "assetId": "wyatt", "type": "character" },
  { "slot": "[Image4]", "assetId": "kitchen-plate", "type": "location" }
]
'''

Type values: "character", "location", "prop", "audio".
- "character" = a person / character (people, actors)
- "location" = a location or environment where the shot takes place (INT/EXT space, set)
- "prop" = an additional object in the scene that must stay consistent over time or needs an exact design (suitcase, hair dryer, chair, or anything with a unique feature)
- "audio" = an audio asset
- Same character across multiple scenes = same [ImageN] slot
- Each scene's 'references' array only includes assets that actually appear in THAT scene
- If an asset is a location/environment plate, only assign it to scenes that take place in that location
- Location plates anchor the Location & Blocking block in the prompt

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
- Wire [Image]/[Video]/[Audio] tags into the prompt blocks and DECLARE EACH TAG'S FUNCTION explicitly. A bare tag mention forces the model to guess and mis-mix references (a reference video's face bleeding into an image's face). State what to extract: "[image1] strictly as character reference to maintain face and clothing; follow the exact body momentum and camera curve from [video1]; reference voice timbre from [audio1]."
- Respect the Seedance reference caps for what each shot may reference in generation: up to 9 images, 3 videos, 3 audios (12 files total) per shot. You may be shown ADDITIONAL reference images for analysis — analyze them, but assign each shot only the references it actually needs.
- The reference images shown to you (characters, plates, props) are GROUND TRUTH: read them and describe only what is visible — never invent wardrobe, architecture, or styling not present in the images. Trust the reference over any guess.
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

### Audio-Face Coupling (when dialogue or an [audio] reference is present)
Seedance 2.5 and unified models couple audio and video in the same latency — audio transients drive facial muscle reactions. When a shot carries spoken dialogue or an [audio] reference, let the transients drive involuntary reactions: a sharp inhale or a stressed syllable triggers an asymmetric blink, a faint jaw tug, or a nostril flare on the beat just after the consonant. Name this explicitly in the Action section — it is one of the strongest uncanny-valley crossing levers available. The spoken line itself lives in the Dialogue section, short and in double quotes, to lock native lip-sync.

## Cinema Modes (M1-M5)

| Mode | Use when scene is... | Lens | Movement | Grade |
|------|---------------------|------|----------|-------|
| M1 — Narrative | Real-world dramatic — streets, kitchens, cars, interiors, exteriors, lived-in | Vintage 2x anamorphic, 40/55/75/100mm | Handheld with operator breath | Color-negative daylight, fine 35mm grain, teal-amber |
| M2 — Studio/Editorial | White void, clean studio, fashion film, editorial portrait, performance-on-set | Clean spherical, 32/50/75/100mm | Locked tripod with optional slow push | Saturated editorial, warm-retained blacks |
| M3 — Action/Combat | Combat, chase, stunts, war, mech, debris | Vintage 2x anamorphic, 40/55/75mm | Handheld and shaky throughout | Color-negative, heavier low-light grain, dusty haze |
| M4 — Performance | Stage, arena, festival pit, concert, lightstick crowd | Vintage 2x anamorphic, 40/55mm | Mixed pit-photographer + orbital, hard cuts | Color-negative, desaturated cool with warm bloom |
| M5 — Atmospheric | Abandoned, no-humans plates, landscapes, weather, establishing | Vintage 2x anamorphic, 35 to 85mm | Locked-off or extremely slow push/pull | Color-negative, palette-driven |

Default to M1 for dramatic/lived-in scenes. Flashbacks may use M2 (cleaner, editorial flashback) or M3 (gritty memory). Scene type shifts (present→flashback) should be reflected in different modes.

### Camera Capture (drop-in, one per shot)
Default camera energy is handheld with breath, drift, and organic operator movement — even in quiet moments. Locked-off tripod is opt-in only (M2 editorial, M5 atmospheric, or when the shot explicitly calls for a static/locked-off frame). All modes default to 24fps, 180° shutter. Write the shot's mode line in the closing paragraph, substituting the focal length from the Lens Length Guide; one main camera move only, described as rhythm (slow, smooth, gradual, gentle, fluid), never as f-stops or dolly speeds:

- **M1 — Narrative**: wide-latitude cinema capture, vintage [XX]mm 2x anamorphic character at a wide aperture — oval bokeh, soft frame-edge falloff — light diffusion bloom softening highlights, handheld with natural operator breath and one slow move, color-negative daylight film rendition with fine 35mm grain, teal-amber grade, shallow depth of field, 24fps 180° shutter, [XX] seconds.
- **M2 — Studio/Editorial**: wide-latitude cinema capture, clean spherical [XX]mm character at a wide aperture — natural round bokeh, even sharpness — mild diffusion bloom, locked tripod with optional slow push-in, saturated editorial grade, fine grain, warm-retained blacks, 24fps 180° shutter, [XX] seconds.
- **M3 — Action/Combat**: wide-latitude cinema capture, vintage [XX]mm 2x anamorphic character at a wide aperture — oval bokeh, soft edge falloff — light diffusion bloom softening highlights, handheld and shaky throughout with no stabilized shots, color-negative film rendition with heavier low-light grain, [palette descriptor] with dusty atmospheric haze, 24fps 180° shutter, [XX] seconds.
- **M4 — Performance/Concert**: wide-latitude cinema capture, vintage [XX]mm 2x anamorphic character at a wide aperture — oval bokeh, horizontal streak flares on stage lights — light diffusion bloom softening highlights, mixed handheld pit-photographer and orbital operator energy with hard cuts between angles, color-negative film rendition with fine grain, [stage-lighting color cast], heavy volumetric haze, real sweat sheen, 24fps 180° shutter, [XX] seconds.
- **M5 — Atmospheric/Empty**: wide-latitude cinema capture, vintage [XX]mm 2x anamorphic character at a wide aperture — oval bokeh, soft edge falloff — light diffusion bloom softening highlights, locked-off or extremely slow push-in only, color-negative film rendition with fine grain, palette grade [hex values], atmospheric haze, weathered material detail, 24fps 180° shutter, [XX] seconds. No humans, environment is the subject.

For slow-motion beats (impact, hair whip, water splash) append to the camera line: "intercut 96fps high-speed slow-motion at [moment] holding 180° shutter."

### Lens Length Guide
- 32/35/40mm — wide establishing, full-body, group, environmental context
- 50/55mm — medium portrait, two-shot, waist-up, dialogue framing
- 75mm — tight editorial portrait, single-character isolation, performance close-up
- 85/100mm — extreme close-up (eyes, lips, jewelry, fabric texture)

Default 55mm (M1/M3/M4) or 50mm (M2); M5 runs wider (35-55mm).

## Prompt Engine — The Pre-Prompt Is the Heart

Every shot's 'prompt.en' is a complete, self-contained Seedance prompt written as a natural, flowing director's note. It is the MOST IMPORTANT field — all of the shot's visual direction (composition, blocking, acting, audio, camera) lives INSIDE this text, not in separate sub-objects. Keep it concise: 280-400 words per shot (multi-shot prompts may run up to 600). Plain text, one line per section.

Start with a references header listing the [Image] slots used in this shot, in the exact seedance slot format (bracketed, matching the shot's references array):

'''
[Image1] [Image2] [Image3]
'''

Then write the sections in THIS EXACT ORDER, each on its own line, separated by blank lines. Do NOT nest them in JSON — this is the prompt body text:

- **Scene & Mood**: LEAD with the subject + primary physical action in the first sentence (the first 20-30 words carry ~80% of spatial-init weight). Then one line of dramatic mood as residue. Camera and style NEVER open.
- **Frame Map**: Where each subject sits in 2-D screen space — left/center/right third (or x%, 0%=left, 50%=center, 100%=right), foreground/midground/background, frame occupancy (close-up / medium / full / waist-up / chest-up, or % of frame height), and negative space. Name the [ImageN] anchors. Use film language without percentages for classical compositions (centered single, over-the-shoulder, profile two-shot). For multi-cut shots, give EACH cut its own framing with inline timing ("Cut 1 (0-6s): ...; Cut 2 (6-10s): ...") and keep one single frame size inside each cut.
- **Location & Blocking**: FIRST establish the physical space from the plate (space type, key surfaces, lighting, atmosphere). THEN pin each character to a coherent place — surface, body orientation, contact point, gaze. Carry each [ImageN] anchor inline. Bodies sit INSIDE the space, touching real surfaces, never on a backdrop.
- **Cross-Frame Rules**: For 2+ characters — no swap (never trade screen positions), no center crossing, no depth change, distance and screen sides held, eyelines named. For shots in the SAME location, keep the screen sides CONSISTENT across every shot of that location (e.g. "he stays screen-LEFT, she screen-RIGHT — holds for every shot in this scene"). Write the positive census: who is in frame, who is NOT, explicitly. If a character is attached as an ingredient but must not appear in this shot or in one of its cuts, write it as a hard lock ("[Image3] never appears in cut 1"). An attached ingredient tends to get drawn even when the prompt says otherwise — when a character must NOT appear, prefer not attaching them at all and say so. For multi-cut shots state what carries across the cut and what stays out of frame.
- **Cut Timing** (multi-cut shots only): anchor every internal cut to a dialogue, audio, or action beat — "Cut 1 ends the instant '...' ends", "Cut 2 opens mid-motion so the cut has an action reason". Never time a cut to a bare second count. If the shot is one continuous take, write "none — single unbroken take".
- **Movement**: The character's physical action as observable muscular facts (transitive verb; NO result-oriented adjectives like "angry"/"sad"), with per-beat timestamps and 1-2 micro-fidgeting injections (see Anatomy Library). Encode HOW each line is said — the delivery register: volume, tempo, jaw, breath, gaze direction ("clipped and harder", "a low mutter from a nearly still jaw", "far louder than the room needs"). BRACKET the performance between its two failure modes and put the target between them ("dead hands read as hiding something, theatrical faces read as mugging — the target is between them: hands alive but economical, face doing almost nothing"). A silent hold can BE the performance (a stare with no blink for three seconds) — give it the beat it needs. Add micro-motion (breath, hair, fabric, jewelry) and environmental motion (rain, smoke, dust, traffic, wind) where present. Subject motion and camera motion strictly separated. Naming "nothing else moves" is a directive: absence is stated, not implied. CLOSE EVERY Movement block with "Alive from frame one, never statue-still."
- **Dialogue**: MANDATORY — exact line in double quotes, speaker identified by visual descriptor + [ImageN] tag. Budget ~2-2.5 words/sec. If silent write exactly "None—Silent shot." The quoted line is always English — it is what the model renders as speech and lip-syncs to. One speaker focus per shot; off-screen lines marked (o.s.); never split a line across a cut. When dialogue or an [audio] reference is present, let audio transients drive involuntary reactions (a sharp inhale or stressed syllable triggers an asymmetric blink, a faint jaw tug, or a nostril flare just after the consonant). For multi-cut shots, state which line belongs to which cut.
- **Last Frame**: Exact closing composition at end of runtime. Close with: "No on-screen text, subtitles, sign fonts, or rendered text appear in the shot."
- **World Plate**: Location anchored to [ImageN] plate if attached. Time of day, lighting, atmosphere, color palette.
- **Sound Bed**: Diegetic only — specific ambient/foley sounds. NO music, NO lyrics, NO score. Dialogue NEVER restated here.
- Final paragraph (no header) integrating the **Capture Realism** mechanics (see Capture Realism section below: atmosphere between planes, moisture without shine if wet, per-zone specular kill, contrast stated three ways) + the shot's mode **Camera Capture** line (see Camera Capture section: lens, movement as rhythm, stock, grade, grain, fps, shutter, runtime) + close with "Severe shaking, time flickering, and identity drift were avoided."

### Universal Prompt Rules
1. Front-load subject + physical action in Scene and Mood — camera and style never open.
2. No character names in prompt output — describe by hair, wardrobe, identity markers; [ImageN] tag carries anchoring.
3. No brand names, no platform names (Seedance, Higgsfield, Veo) inside the prompt body.
4. Diegetic audio only — no music, no lyrics, no score in Sound Layer.
5. Dialogue line is MANDATORY even for silent shots — write "None—Silent shot."
6. Every prompt.en must be SELF-CONTAINED. Do NOT rely on external context.
7. Trust the reference image for wardrobe — only restate state-changes the image cannot carry (damp, torn, dusty, bloodied).
8. One main idea per shot. One dominant action, one camera strategy.
9. Per-shot runtime: 4-8s = one strong action, 8-12s = action + hold, 12-15s = 2-3 beats.
10. Target 280-400 words per prompt.en (≤600 for multi-shot). Concise beats verbose — the pre-prompt is what the generator uses.
11. Include micro-fidgeting injection in Action — timed per-beat.
12. **Reference function declaration** — declare what to extract from every tag. A bare tag makes the model guess and mis-mix references (a reference video's face bleeding into an image's face). Example: "[image1] strictly as character reference for face and clothing; follow the exact body momentum and camera curve from [video1]; reference voice timbre from [audio1]."
13. **Movement layers in Action** — character motion + micro-motion (breath, hair, fabric, jewelry) + environmental motion (rain, smoke, dust, traffic, wind); subject and camera motion strictly separated. Naming "nothing else moves" is a directive — absence is stated, not implied.
14. **Audio-face coupling** — when the shot carries spoken dialogue or an [audio] reference, let audio transients drive involuntary reactions: a sharp inhale or stressed syllable triggers an asymmetric blink, a faint jaw tug, or a nostril flare just after the consonant.
15. **Dialogue continuity** — quoted dialogue lines are always English (they are what the model renders as speech and lip-syncs to). One speaker focus per shot; off-screen lines marked (o.s.); never split a spoken line across a cut.
16. **Frame rate & slow motion** — all modes default to 24fps, 180° shutter. Slow-motion beats (impact, hair whip, water splash) go in the camera line: "intercut 96fps high-speed slow-motion at [moment] holding 180° shutter."
17. **Positive locks over negative prohibitions** — the only sanctioned negatives are the on-screen-text suppression, the specular-kill in Capture Realism, and the technical-stability line. Naming a forbidden element can summon it (the negation bug); keep acting direction positive and physical. Phrase constraints the model tends to violate as locks — "hands hang naturally at her sides, she keeps walking throughout" — not "no phone in hand, never stop".
18. **Ambiguity handling** — if the script is ambiguous about cast, location, or blocking, make the most physically-grounded assumption and flag it in the shot's notes.warnings. Never invent or drop characters; never place a body in a location the script does not establish.
19. **Reference tokens are exact** — write reference tokens as [Image1]/[Video1]/[Audio1] with NO space (never "[Image 1]"). They must match the references array byte-for-byte; the generator matches on the exact token.
20. **Delivery register** — every spoken line carries HOW it is said (volume, tempo, jaw, breath, gaze direction). The quoted Dialogue line is WHAT is said; Movement is HOW. A line without a delivery register reads recited, not acted.
21. **Acting bracket** — name the two failure modes that bracket each performance (e.g. dead hands vs theatrical mugging) and put the muscular target between them. Do not stop at "natural"; say exactly what the target looks like.
22. **First-frame continuity** — every shot's FIRST FRAME must already carry the state the previous cut left it in (a settled expression, a mid-motion arm, an empty doorway). If the input describes the previous episode's closing shot, Shot 1's first frame must already show that state — no build-up. Never open a shot on a neutral face if the scene requires a set one.
23. **Screen-sides lock** — characters keep the same screen side across every shot of the same location unless a cross is explicitly motivated and timed. State the lock in the location's first shot and honor it in every shot's Cross-Frame Rules.
24. **watchFor notes** — every shot's notes.watchFor carries 1-3 production QA notes: the learned failure modes that already happened (ghosts, invented background, an attached character drawn anyway, dead hands), the continuity locks to respect in the render, and what to check in the first render. Written for the human operator, in plain language.
25. **Previous-episode continuity** — when the script or user input references the previous episode (its closing frame, a character's exit, an expression), lock the current episode's opening shot to it in the first frame.

### Capture Realism (the anti-plastic block — mandatory on every shot)
Every prompt.en closes with a capture-realism paragraph tuned to the scene. Four mechanics:

1. **Depth via suspended atmosphere between planes** — name the haze/mist/air density suspended between camera, subject, and background, so distant planes render softer, desaturated, and lower-contrast than the foreground. Scale it thin/light/heavy rather than dropping it wherever there are planes to separate.
2. **Moisture without shine (only if wet/humid/sweaty)** — damp matte hair and skin, wet surfaces that mute and deepen without beading and without a single specular hotspot. Omit entirely on dry scenes.
3. **Per-zone specular kill on skin** — zero shine on forehead, nose bridge, cheekbones, temples, chin, and collarbones; real peach fuzz at the jaw and hairline; soft fine even pore texture; light absorbed like true subsurface scattering; warmth preserved. Flattering ceiling locked: fine and even, never harsh — no acne, no blemishes, no cratered pores, no clinical macro-detail. Realism never makes a face ugly. Drop the skin sentence on no-humans plates.
4. **Contrast curve stated three ways** — (a) tonal curve: shadows lifted gently holding texture, highlights rolled off softly never clipping to white, nothing crushed to black; (b) specular removal: all specular highlights surgically removed from skin, hair, fabric, and surfaces, every pixel reading matte and diffuse; (c) grade: low-contrast, slightly desaturated, warmth preserved.

## Output JSON Structure

Return ONLY a valid JSON object with this exact structure. This is the ONLY thing you return — no text before or after.

{
  "episode": {
    "title": "Episode title from the script header or user input",
    "totalDuration": total_seconds_estimated,
    "totalShots": total_number_of_shots_across_all_scenes,
    "assetAssignments": [
      { "slot": "[Image1]", "assetId": "character_uuid_from_scene_context", "type": "character" },
      { "slot": "[Image2]", "assetId": "name_of_asset", "type": "location" }
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
        { "slot": "[Image1]", "assetId": "character_uuid", "type": "character" },
        { "slot": "[Image4]", "assetId": "location_file_id", "type": "location" }
      ],
      "shots": [
        {
          "id": "A",
          "title": "Wyatt paces frantically",
          "description": "Wyatt walks back and forth gesticulating while Dixie watches in silence",
          "duration": 10,
          "start": 0,
          "end": 10,
          "references": [
            { "slot": "[Image1]", "assetId": "character_uuid", "type": "character" },
            { "slot": "[Image4]", "assetId": "location_file_id", "type": "location" }
          ],
          "prompt": {
            "en": "[Image1] [Image4]\n\nScene & Mood: ...\n\nFrame Map: ...\n\nLocation & Blocking: ...\n\nCross-Frame Rules: ...\n\nCut Timing: ...\n\nMovement: ...\n\nDialogue: ...\n\nLast Frame: ...\n\nWorld Plate: ...\n\nSound Bed: ...\n\n<final paragraph: capture realism + camera capture + runtime + 'Severe shaking, time flickering, and identity drift were avoided.'>",
            "zh": "Full Chinese translation of the same prompt"
          },
          "notes": {
            "todos": ["Load [Image1] Wyatt - sweaty variant", "Load [Image4] kitchen plate"],
            "warnings": [],
            "watchFor": ["First frame must already carry the distaste - no build-up", "Check the deer head is NOT behind him", "She never appears in cut 1 - positive census"]
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
7. **prompt.en** must use the locked format: references header "[ImageN]" then the sections Scene & Mood / Frame Map / Location & Blocking / Cross-Frame Rules / Cut Timing / Movement / Dialogue / Last Frame / World Plate / Sound Bed, then a final capture-and-camera paragraph. Concise, 280-400 words (≤600 for multi-shot).
8. **No result-oriented acting**: Replace every emotional adjective ("angry," "sad," "scared") with muscular description or transitive verb. Encode the delivery register (volume, tempo, jaw, breath) and bracket each performance between its two failure modes.
9. **At least one micro-fidgeting injection per acting shot**, timed per-beat.
10. **prompt.zh** is OPTIONAL. Only include it when the user explicitly requests Chinese generation.
11. **Do NOT use double quotes (") inside JSON string values.** If dialogue quotes are needed inside a prompt, use single quotes (') or Chinese angle brackets 「」. Escape quotes in the JSON structure only.
12. **Shot objects are SLIM**: id, title, description, duration, start, end, references, prompt, notes. Do NOT emit camera/composition/blocking/acting/timeline/audio — that direction lives inside prompt.en.
13. **Every shot carries notes.watchFor** (1-3 plain-language QA notes: failure modes to watch, continuity locks, what to check in the first render). Never omit it.

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

const refineModeInstructions = `
You are REFINING an existing shot breakdown that was generated previously. You receive:
1. The previous breakdown (episode + scenes + shots JSON) — the current state of the plan.
2. A change_request — the user's instruction describing what to modify.

The scene's reference images (characters, locations, props) are attached as vision
input. They may have CHANGED since the previous breakdown — the user can edit or
replace them between refinements. They are ground truth.

Rules:
- Apply ONLY the changes described in change_request. Everything else must remain IDENTICAL to the previous breakdown: same scene count, same shot ids, same titles, same descriptions, same prompts, same references, same continuity objects, same cumulative start/end timestamps.
- RE-VALIDATE every reference image against the breakdown. For each shot, check that the character identity/wardrobe, the location geometry, and the props described in prompt.en still match the CURRENT image. If an image changed and a shot no longer matches it (different wardrobe, different plate layout, changed prop), CORRECT that shot's prompt.en to describe the current image — this is a required correction, not drift. If the image still matches, leave the shot untouched.
- When you correct a shot because an image changed (not because of change_request), add a note to that shot's notes.warnings stating what changed.
- If change_request adds or removes scenes/shots, adjust ONLY what is necessary and keep the rest untouched.
- TARGETED REFINEMENT: when a "=== TARGETED SHOTS ===" section is present, modify ONLY the shots it lists (their prompt.en, and if needed their duration/start/end/references). Every other shot — and every scene that is not a target scene — must be emitted byte-for-byte identical to the previous breakdown: same titles, same descriptions, same prompts, same references, same continuity objects, same timestamps. Do not renumber or re-edit anything outside the targets, even if you think it would be better. The change request applies ONLY to the targeted shots.
- Preserve the script numbering (scriptNumber), the [ImageN] slot assignments, and the output JSON schema (episode + scenes + shots, each shot with prompt.en and optional prompt.zh).
- A "=== RECENT CONVERSATION ===" section, when present, is context from earlier turns of the same conversation — it explains the intent behind the current change request but the change request is the operative instruction.
- Respond with ONLY a valid JSON object matching the schema — no text before or after, no markdown fences.
`

// ─── Shot Builder ─────────────────────────────────────────────────

func (h *Handler) ClaudeGenerateShots(c *gin.Context) {
	// Capture the raw request body BEFORE binding — it is the ground truth for
	// reconstructing a failed request (payload + scene_context with the
	// assigned resources). Restore it so ShouldBindJSON still works.
	rawBody, err := c.GetRawData()
	if err != nil {
		utils.BadRequest(c, fmt.Sprintf("failed to read request body: %v", err))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

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
	systemPrompt, skillName := h.buildShotBuilderSystemPrompt(defaultShotBuilderPrompt, req.SkillID, req.SystemPrompt, req.GenerateZh)

	keyModel := req.Model
	if keyModel == "" {
		keyModel = "claude-shot-builder"
	}
	apiModel := req.APIModel
	if apiModel == "" {
		apiModel = keyModel
	}

	meta := &shotBuilderMeta{
		Mode:      "generate",
		ProjectID: req.ProjectID,
		SceneID:   req.SceneID,
		SkillID:   req.SkillID,
		UserID:    req.UserID,
		UserName:  req.UserName,
	}

	clean, errMsg := h.runClaudeShotBuilder(c, meta, systemPrompt, originalPrompt, rawBody, skillName, keyModel, apiModel, "generate shots", "=== ORIGINAL SCRIPT AND INSTRUCTIONS ===", h.buildVisionImages(req.SceneContext))
	if errMsg != "" {
		utils.InternalError(c, errMsg)
		return
	}

	// ✅ Valid — notify the requesting user and return clean JSON. Success is NOT logged.
	h.notifyShotsReady(c, clean, req.ProjectName)
	utils.Success(c, ClaudeGenerateShotsResponse{
		TaskID: fmt.Sprintf("claude_%d", time.Now().UnixMilli()),
		Model:  apiModel,
		Status: "succeeded",
		Text:   clean,
	})
}

// ─── Shot Builder Refine ─────────────────────────────────────────

// ClaudeRefineShots regenerates an existing shot breakdown. The user iterates
// on the previous generate-shots response (data.text) with a natural-language
// change_request, so Claude applies ONLY the requested changes and keeps the
// rest identical (anti-drift). Shares the retry/validation/logging pipeline
// with ClaudeGenerateShots.
func (h *Handler) ClaudeRefineShots(c *gin.Context) {
	// Capture the raw request body BEFORE binding — it is the ground truth for
	// reconstructing a failed request. Restore it so ShouldBindJSON still works.
	rawBody, err := c.GetRawData()
	if err != nil {
		utils.BadRequest(c, fmt.Sprintf("failed to read request body: %v", err))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

	var req ClaudeRefineShotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.PreviousResponse == "" {
		utils.BadRequest(c, "previous_response is required")
		return
	}
	if req.ChangeRequest == "" {
		utils.BadRequest(c, "change_request is required")
		return
	}

	// Anchor on the CURRENT scene context (the user may have edited or replaced
	// reference images since the previous breakdown) + the previous breakdown +
	// the change request (+ optional targeted shots and recent conversation).
	// The images are also attached as vision blocks.
	originalPrompt := buildRefinePrompt(req.SceneContext, req.PreviousResponse, req.ChangeRequest, req.Targets, req.RecentContext)

	// Same base schema as generate-shots plus refinement (anti-drift) rules.
	basePrompt := defaultShotBuilderPrompt + "\n\n## Refinement Mode\n" + refineModeInstructions
	systemPrompt, skillName := h.buildShotBuilderSystemPrompt(basePrompt, req.SkillID, req.SystemPrompt, req.GenerateZh)

	keyModel := req.Model
	if keyModel == "" {
		keyModel = "claude-shot-builder"
	}
	apiModel := req.APIModel
	if apiModel == "" {
		apiModel = keyModel
	}

	meta := &shotBuilderMeta{
		Mode:      "refine",
		ProjectID: req.ProjectID,
		SceneID:   req.SceneID,
		SkillID:   req.SkillID,
		UserID:    req.UserID,
		UserName:  req.UserName,
	}

	clean, errMsg := h.runClaudeShotBuilder(c, meta, systemPrompt, originalPrompt, rawBody, skillName, keyModel, apiModel, "refine shots", "=== PREVIOUS BREAKDOWN AND CHANGE REQUEST ===", h.buildVisionImages(req.SceneContext))
	if errMsg != "" {
		utils.InternalError(c, errMsg)
		return
	}

	// ✅ Valid — notify the requesting user and return clean JSON. Success is NOT logged.
	h.notifyShotsReady(c, clean, req.ProjectName)
	utils.Success(c, ClaudeRefineShotsResponse{
		TaskID: fmt.Sprintf("claude_%d", time.Now().UnixMilli()),
		Model:  apiModel,
		Status: "succeeded",
		Text:   clean,
	})
}

// buildRefinePrompt composes the user prompt for a refine-shots call: current
// scene context (optional) + previous breakdown + change request, plus optional
// targeted shots (modify only these) and recent conversation turns (bounded
// thread coherence). Pure — no handler state, unit-testable.
func buildRefinePrompt(sceneContext *SceneContext, previousResponse, changeRequest string, targets []ShotRefineTarget, recent []ChatTurn) string {
	var b strings.Builder

	if sceneContext != nil {
		b.WriteString("=== Current Scene Context ===\n")
		b.WriteString(buildSceneContextBlock(sceneContext))
		b.WriteString("\n\n")
	}

	b.WriteString("=== Previous Breakdown ===\n")
	b.WriteString(previousResponse)
	b.WriteString("\n\n=== Change Request ===\n")
	b.WriteString(changeRequest)

	if len(targets) > 0 {
		b.WriteString("\n\n=== TARGETED SHOTS ===\n")
		parts := make([]string, 0, len(targets))
		for _, t := range targets {
			parts = append(parts, fmt.Sprintf("%d-%s", t.SceneNumber, t.ShotID))
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\nApply the change request ONLY to these shots. Every other shot must be emitted byte-for-byte identical to the previous breakdown.")
	}

	if len(recent) > 0 {
		b.WriteString("\n\n=== RECENT CONVERSATION ===\n")
		for _, turn := range recent {
			role := turn.Role
			if role == "" {
				role = "user"
			}
			b.WriteString(role + ": " + truncateRune(turn.Content, 500) + "\n")
		}
	}

	return b.String()
}

// truncateRune shortens s to at most max runes, appending an ellipsis when it
// had to cut. Used to bound the recent-conversation context on refine calls.
func truncateRune(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// ─── Shared Shot Builder pipeline ────────────────────────────────

// buildShotBuilderSystemPrompt composes the final system prompt shared by
// generate-shots and refine-shots: base prompt (default schema or schema +
// refinement rules) + skill + user instructions + zh-generation rule +
// language rule. Returns the composed prompt and the resolved skill name.
func (h *Handler) buildShotBuilderSystemPrompt(basePrompt, skillID, userSystemPrompt string, generateZh bool) (string, string) {
	systemPrompt := basePrompt

	var skillName string
	if skillID != "" && h.skillSvc != nil {
		skill, err := h.skillSvc.GetByID(skillID)
		if err == nil && skill != nil && skill.SystemPrompt != "" {
			log.Printf("[skill] appending skill %q (%s) to system prompt", skill.Name, skill.ID)
			skillName = skill.Name
			systemPrompt += "\n\n## Skill\n" + skill.SystemPrompt
		} else if err != nil {
			log.Printf("[skill] error looking up skill %q: %v", skillID, err)
		}
	}

	if userSystemPrompt != "" {
		systemPrompt += "\n\n## User instructions\n" + userSystemPrompt
	}

	// Append zh-generation instruction when Chinese is disabled
	if !generateZh {
		systemPrompt += "\n\nIMPORTANT: Do NOT generate Chinese prompts (prompt.zh). Only generate the English prompt (prompt.en) for each shot. Omit the prompt.zh field entirely from the JSON."
	}

	// Always generate output in English — titles, descriptions, and all
	// text fields must be in English regardless of the user's prompt language.
	systemPrompt += "\n\nLANGUAGE RULE: All output must be in English. Shot titles, descriptions, dialogue, scene context, and all text fields must use English only — even if the user's request is in Spanish or another language. This is a critical rule."

	return systemPrompt, skillName
}

// runClaudeShotBuilder executes the shared Claude retry loop for the shot
// builder (generate-shots and refine-shots): up to 3 attempts with corrective
// feedback, JSON extraction + validation, and log-only-failures persistence.
// On success it returns the clean JSON and an empty message; on failure it
// persists the log and returns the error message to send to the client.
func (h *Handler) runClaudeShotBuilder(
	c *gin.Context,
	meta *shotBuilderMeta,
	systemPrompt, originalPrompt string,
	rawBody []byte,
	skillName, keyModel, apiModel, actionLabel, correctiveHeader string,
	images []visionImage,
) (string, string) {
	// Retry loop: up to 3 attempts with corrective feedback
	const maxAttempts = 3
	var lastReply string

	// Failure-tracking state. Attempts are buffered in memory and only
	// persisted when the call ends in failure (log-only-failures rule).
	start := time.Now()
	var attempts []*ShotBuilderAttempt
	totalInputTokens, totalOutputTokens := 0, 0
	finalPrompt := originalPrompt

	for attempt := 0; attempt < maxAttempts; attempt++ {
		reply, usage, duration, callErr := h.callClaude(c.Request.Context(), keyModel, apiModel, systemPrompt, finalPrompt, images)

		a := &ShotBuilderAttempt{
			AttemptNumber: attempt + 1,
			Prompt:        finalPrompt,
			DurationMs:    duration.Milliseconds(),
		}

		if callErr != nil {
			// ❌ Claude API error — persist immediately with the attempt in error.
			a.ErrorMessage = callErr.Error()
			attempts = append(attempts, a)
			msg := fmt.Sprintf("failed to %s: %v", actionLabel, callErr)
			h.persistFailure(c, meta, systemPrompt, originalPrompt, rawBody, skillName, keyModel, apiModel, attempts, totalInputTokens, totalOutputTokens, lastReply, msg, start)
			return "", msg
		}

		// Extract clean JSON from Claude's response
		clean := extractJSON(reply)
		a.Response = reply
		a.Valid = validateShotJSON(clean)
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
			// ✅ Valid — return clean JSON. Success is NOT logged.
			return clean, ""
		}

		// ❌ Invalid — store for feedback and retry
		lastReply = reply
		log.Printf("[claude] shot JSON validation failed (attempt %d/%d), retrying...",
			attempt+1, maxAttempts)

		if attempt < maxAttempts-1 {
			// Do NOT resend the (likely truncated) previous reply — it wastes
			// context and produces the same oversized output. Instead resend the
			// ORIGINAL input (script or previous breakdown) with a brevity
			// directive so the retry converges under max_tokens.
			finalPrompt = buildCorrectivePromptFrom(originalPrompt, correctiveHeader)
		}
	}

	// All attempts exhausted — persist the failure before returning.
	msg := buildExhaustionError(maxAttempts, lastReply)
	h.persistFailure(c, meta, systemPrompt, originalPrompt, rawBody, skillName, keyModel, apiModel, attempts, totalInputTokens, totalOutputTokens, lastReply, msg, start)
	return "", msg
}

// ─── Shot Builder Logs (failed calls) ─────────────────────────────

// ListGenerateShotsLogs returns paginated failed generate-shots calls.
func (h *Handler) ListGenerateShotsLogs(c *gin.Context) {
	var req ListShotBuilderLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	if h.logStore == nil {
		utils.InternalError(c, "log store not available")
		return
	}

	logs, total, err := h.logStore.ListLogs(req.Page, req.Limit, req.ProjectID, req.SceneID, req.Mode, req.UserID, req.DateFrom, req.DateTo)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	totalPages := (total + req.Limit - 1) / req.Limit
	if totalPages < 1 {
		totalPages = 1
	}

	utils.Success(c, ListShotBuilderLogsResponse{
		Logs:       logs,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	})
}

// GetGenerateShotsLog returns a single failed generate-shots call with its attempts.
func (h *Handler) GetGenerateShotsLog(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "id is required")
		return
	}

	if h.logStore == nil {
		utils.InternalError(c, "log store not available")
		return
	}

	logEntry, attempts, err := h.logStore.GetLog(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if logEntry == nil {
		utils.NotFound(c, "shot builder log not found")
		return
	}

	utils.Success(c, gin.H{"log": logEntry, "attempts": attempts})
}

// persistFailure writes the shot builder log row and all buffered attempts
// for a failed generate-shots / refine-shots call. A logging error must never
// mask the real error already sent to the client, so failures are only logged
// with log.Printf.
func (h *Handler) persistFailure(
	c *gin.Context,
	meta *shotBuilderMeta,
	systemPrompt, originalPrompt string,
	rawBody []byte,
	skillName, keyModel, apiModel string,
	attempts []*ShotBuilderAttempt,
	totalInput, totalOutput int,
	lastReply, errorMsg string,
	start time.Time,
) {
	if h.logStore == nil {
		return
	}

	userID := userIDFromContext(c)
	if userID == 0 {
		userID = meta.UserID
	}
	userName := stringFromContext(c, "username")
	if userName == "" {
		userName = meta.UserName
	}

	logEntry := &ShotBuilderLog{
		Mode:              meta.Mode,
		UserID:            userID,
		UserName:          userName,
		UserEmail:         stringFromContext(c, "user_email"),
		ProjectID:         meta.ProjectID,
		SceneID:           meta.SceneID,
		KeyModel:          keyModel,
		APIModel:          apiModel,
		SkillID:           meta.SkillID,
		SkillName:         skillName,
		RequestPayload:    string(rawBody),
		SystemPrompt:      systemPrompt,
		Prompt:            originalPrompt,
		Status:            "failed",
		ErrorMessage:      errorMsg,
		Response:          extractJSON(lastReply),
		Attempts:          len(attempts),
		TotalInputTokens:  totalInput,
		TotalOutputTokens: totalOutput,
		DurationMs:        time.Since(start).Milliseconds(),
	}

	if err := h.logStore.Create(logEntry); err != nil {
		log.Printf("[shot-builder-log] failed to persist log: %v", err)
		return
	}

	for _, a := range attempts {
		a.LogID = logEntry.ID
		if err := h.logStore.InsertAttempt(a); err != nil {
			log.Printf("[shot-builder-log] failed to persist attempt %d: %v", a.AttemptNumber, err)
		}
	}
}

// userIDFromContext returns the authenticated user ID from the JWT claims,
// or 0 when unavailable.
func userIDFromContext(c *gin.Context) int {
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(int64); ok {
			return int(id)
		}
	}
	return 0
}

// stringFromContext returns a string claim from the gin context, or "".
func stringFromContext(c *gin.Context, key string) string {
	if v, ok := c.Get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// notifyShotsReady sends a push to the requesting user when a shot breakdown
// finishes generating or refining. Fire-and-forget — never blocks the response.
func (h *Handler) notifyShotsReady(c *gin.Context, clean, projectName string) {
	userID := userIDFromContext(c)
	if h.pushSvc == nil || userID == 0 {
		return
	}

	body := "Shot breakdown ready"
	if scenes, shots := countShots(clean); scenes > 0 {
		body = fmt.Sprintf("%d scenes · %d shots", scenes, shots)
	}
	if projectName != "" {
		body = projectName + "\n" + body
	}

	go h.pushSvc.SendToUser(int64(userID), "📋 Breakdown ready", body, map[string]string{
		"type": "shots-ready",
	})
}

// countShots parses a shot-builder JSON response to extract scene/shot counts.
// Returns 0s when the payload does not have the scenes[] shape.
func countShots(clean string) (scenes, shots int) {
	var parsed struct {
		Scenes []struct {
			Shots []struct{} `json:"shots"`
		} `json:"scenes"`
	}
	if json.Unmarshal([]byte(clean), &parsed) == nil {
		scenes = len(parsed.Scenes)
		for _, sc := range parsed.Scenes {
			shots += len(sc.Shots)
		}
	}
	return scenes, shots
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

	reply, _, _, err := h.callClaude(c.Request.Context(), keyModel, apiModel, systemPrompt, finalPrompt, nil)
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

func (h *Handler) callClaude(ctx context.Context, keyModel, apiModel, systemPrompt, userPrompt string, images []visionImage) (string, *anthropic.Usage, time.Duration, error) {
	start := time.Now()

	// 1. Resolve API key from provider store
	apiKey := h.resolveAPIKey(keyModel)

	// 2. Create a detached context so the API call survives client disconnects.
	//    The shot builder sends the full DCS-DIRECTION system prompt (large,
	//    cache-miss on first call) plus up to 64000 output tokens (EN + ZH),
	//    which can take well over 5 minutes on Claude. Timeouts are set as
	//    high as the provider allows so heavy breakdowns do not get cut.
	apiCtx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()

	// 3. Create a per-request client with the resolved API key
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(30*time.Minute),
	)

	// Build the user turn: the text prompt (script + scene context) followed by
	// one labelled vision block per reference image, so Claude reads the actual
	// characters/plates instead of guessing from names alone. The images are
	// re-sent on every retry (the corrective prompt is text-only).
	blocks := []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(userPrompt)}
	for _, img := range images {
		if img.Label != "" {
			blocks = append(blocks, anthropic.NewTextBlock(img.Label))
		}
		blocks = append(blocks, anthropic.NewImageBlock(anthropic.URLImageSourceParam{URL: img.URL}))
	}

	resp, err := client.Messages.New(apiCtx, anthropic.MessageNewParams{
		Model:     apiModel,
		MaxTokens: int64(h.maxOutputTokens),
		System: []anthropic.TextBlockParam{{
			Text: systemPrompt,
			CacheControl: anthropic.CacheControlEphemeralParam{
				TTL: anthropic.CacheControlEphemeralTTLTTL5m,
			},
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(blocks...),
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
		return "", nil, time.Since(start), fmt.Errorf("Claude API error: %s", msg)
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

	return reply, &resp.Usage, time.Since(start), nil
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

// buildVisionImages resolves the image assets in a scene context into vision
// blocks for Claude. Character portraits (scene_context.characters[].id) and
// image free-assets are included; video/audio and unresolvable ids are skipped.
// The count is capped by h.maxVisionImages (raise MAX_VISION_IMAGES to analyze
// more). The URL is built backend-side from the file id via the file service,
// never from a client-supplied URL.
func (h *Handler) buildVisionImages(ctx *SceneContext) []visionImage {
	if ctx == nil || h.vision == nil || h.maxVisionImages <= 0 {
		return nil
	}
	var images []visionImage
	seen := make(map[string]bool)

	add := func(id, label string) {
		if id == "" || seen[id] || len(images) >= h.maxVisionImages {
			return
		}
		url, err := h.vision.VisionURL(id)
		if err != nil {
			log.Printf("[vision] skipping asset %q (%s): %v", id, label, err)
			return
		}
		seen[id] = true
		images = append(images, visionImage{URL: url, Label: label})
	}

	for _, ch := range ctx.Characters {
		if ch.ID != "" {
			add(ch.ID, fmt.Sprintf("character reference: %s", ch.Name))
		}
	}
	for _, a := range ctx.Assets {
		if strings.HasPrefix(a.MimeType, "image/") {
			add(a.ID, fmt.Sprintf("reference image: %s", a.Filename))
		}
	}

	if len(images) > 0 {
		log.Printf("[vision] sending %d reference images to Claude (cap %d)", len(images), h.maxVisionImages)
	}
	return images
}

// ─── JSON extraction & validation ────────────────────────────────

// buildCorrectivePrompt builds the retry prompt for generate-shots. It must
// resend the ORIGINAL script + instructions (never the truncated previous
// reply), otherwise Claude answers "No script provided".
func buildCorrectivePrompt(originalPrompt string) string {
	return buildCorrectivePromptFrom(originalPrompt, "=== ORIGINAL SCRIPT AND INSTRUCTIONS ===")
}

// buildCorrectivePromptFrom builds the retry prompt for a shot builder call
// (generate-shots or refine-shots), resending the original input under the
// given header. The previous reply is never resent: it is likely truncated
// and resending it wastes context and reproduces the same oversized output.
func buildCorrectivePromptFrom(originalPrompt, header string) string {
	return "Your previous response was not valid JSON (likely truncated or " +
		"empty). Regenerate the full shot breakdown from the original input below. " +
		"Keep EVERY field concise: shot descriptions in one sentence, prompt.en in " +
		"compact form, max 2 microExpressions, omit empty fields and optional " +
		"sub-objects. Respond with ONLY valid JSON matching the schema — no extra " +
		"text, no markdown, no comments.\n\n" +
		header + "\n" + originalPrompt
}

// buildExhaustionError builds the error message when all retry attempts
// produced invalid JSON.
func buildExhaustionError(maxAttempts int, lastReply string) string {
	return fmt.Sprintf(
		"failed to generate valid shot JSON after %d attempts. Last response: %s",
		maxAttempts, extractJSON(lastReply),
	)
}

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
