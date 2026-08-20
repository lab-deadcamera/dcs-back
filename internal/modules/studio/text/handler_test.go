package text

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestPromptNoBackticks verifies the prompt constant has no raw backticks
// that would break Go string syntax.
func TestPromptNoBackticks(t *testing.T) {
	if strings.Contains(defaultShotBuilderPrompt, "`") {
		t.Error("defaultShotBuilderPrompt contains raw backtick character")
	}
}

// TestPromptMentionsEpisode verifies the prompt describes the episode→scenes→shots format.
func TestPromptMentionsEpisode(t *testing.T) {
	required := []string{
		"episode",          // episode-level structure
		"scenes",           // scenes array
		"scriptNumber",     // script number parsing
		"scriptLocation",   // INT/EXT location
		"continuity",       // continuity tracking
		"shots",            // shots inside scenes
		"Scene & Mood",     // locked pre-prompt grammar (pack-style)
		"Frame Map",
		"Location & Blocking",
		"Cross-Frame Rules",
		"Cut Timing",
		"Movement",
		"Dialogue",
		"Last Frame",
		"World Plate",
		"Sound Bed",
		"Severe shaking, time flickering, and identity drift were avoided",
		"micro-fidgeting",  // acting features
		"Anatomical Emotion",
		"delivery register",             // how the line is said (anti-robotic)
		"Alive from frame one, never statue-still",
		"watchFor",                      // per-shot production QA notes
		"Screen-sides lock",
		"First-frame continuity",
	}
	for _, s := range required {
		if !strings.Contains(defaultShotBuilderPrompt, s) {
			t.Errorf("prompt missing required section/keyword: %q", s)
		}
	}
}

// TestPromptHasFormatRule verifies the CRITICAL output format rule is present.
func TestPromptHasFormatRule(t *testing.T) {
	if !strings.Contains(defaultShotBuilderPrompt, "CRITICAL — Output Format") {
		t.Error("prompt missing CRITICAL output format rule")
	}
	if !strings.Contains(defaultShotBuilderPrompt, "valid JSON object") {
		t.Error("prompt missing 'valid JSON object' instruction")
	}
}

// TestExtractJSON tests brace-matching extraction.
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    `{"episode":{},"scenes":[]}`,
			expected: `{"episode":{},"scenes":[]}`,
			desc:     "pure JSON — no text around it",
		},
		{
			input:    `Here is the result: {"episode":{},"scenes":[]}`,
			expected: `{"episode":{},"scenes":[]}`,
			desc:     "text before JSON",
		},
		{
			input:    `{"episode":{},"scenes":[]} Let me know if you need changes.`,
			expected: `{"episode":{},"scenes":[]}`,
			desc:     "text after JSON",
		},
		{
			input:    `no braces here`,
			expected: `no braces here`,
			desc:     "no JSON at all",
		},
		{
			input:    `{unclosed`,
			expected: `{unclosed`,
			desc:     "unclosed brace",
		},
		{
			input:    ``,
			expected: ``,
			desc:     "empty text (API-error path has no reply)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := extractJSON(tc.input)
			if got != tc.expected {
				t.Errorf("\n  got:  %q\n  want: %q", got, tc.expected)
			}
		})
	}
}

// TestValidateShotJSON tests validation of both new and legacy formats.
func TestValidateShotJSON(t *testing.T) {
	tests := []struct {
		json  string
		valid bool
		desc  string
	}{
		{
			json:  `{"episode":{"title":"EP16"},"scenes":[{"shots":[{"id":"A"}]}]}`,
			valid: true,
			desc:  "new format: episode + scenes with shots",
		},
		{
			json:  `{"scenes":[{"shots":[{"id":"A"}]}]}`,
			valid: false,
			desc:  "new format without episode",
		},
		{
			json:  `{"shots":[{"id":"A"}]}`,
			valid: true,
			desc:  "legacy format: flat shots array",
		},
		{
			json:  `{"shots":[]}`,
			valid: false,
			desc:  "empty shots array",
		},
		{
			json:  `{"episode":{},"scenes":[]}`,
			valid: false,
			desc:  "scenes with no shots",
		},
		{
			json:  `not json at all`,
			valid: false,
			desc:  "invalid JSON",
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := validateShotJSON(tc.json)
			if got != tc.valid {
				t.Errorf("\n  input: %s\n  got:   %v\n  want:  %v", tc.json, got, tc.valid)
			}
		})
	}
}

// TestValidateShotJSONCuts verifies per-shot field invariants on cuts.
func TestValidateShotJSONCuts(t *testing.T) {
	tests := []struct {
		json  string
		valid bool
		desc  string
	}{
		{json: `{"shots":[{"id":"A","cuts":2}]}`, valid: true, desc: "integer cuts valid"},
		{json: `{"shots":[{"id":"A","cuts":0}]}`, valid: true, desc: "zero cuts valid (continuous take)"},
		{json: `{"shots":[{"id":"A","cuts":-1}]}`, valid: false, desc: "negative cuts invalid"},
		{json: `{"shots":[{"id":"A","cuts":1.5}]}`, valid: false, desc: "fractional cuts invalid"},
		{json: `{"shots":[{"id":"A","cuts":"2"}]}`, valid: false, desc: "string cuts invalid"},
		{json: `{"shots":[{"id":"A","cuts":null}]}`, valid: false, desc: "null cuts invalid"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			if got := validateShotJSON(tc.json); got != tc.valid {
				t.Errorf("\n  input: %s\n  got:   %v\n  want:  %v", tc.json, got, tc.valid)
			}
		})
	}
}

// TestValidateShotJSONPrompt verifies prompt.en invariants.
func TestValidateShotJSONPrompt(t *testing.T) {
	tests := []struct {
		json  string
		valid bool
		desc  string
	}{
		{json: `{"shots":[{"id":"A","prompt":{"en":"Scene & Mood:\n\nFrame Map:"}}]}`, valid: true, desc: "non-empty prompt.en valid"},
		{json: `{"shots":[{"id":"A","prompt":{"en":"  "}}]}`, valid: false, desc: "blank prompt.en invalid"},
		{json: `{"shots":[{"id":"A","prompt":{"en":"Use @image1 as anchor"}}]}`, valid: false, desc: "@imageN forbidden"},
		{json: `{"shots":[{"id":"A","prompt":{"en":"Use [Image1] as anchor"}}]}`, valid: true, desc: "[ImageN] allowed"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			if got := validateShotJSON(tc.json); got != tc.valid {
				t.Errorf("\n  input: %s\n  got:   %v\n  want:  %v", tc.json, got, tc.valid)
			}
		})
	}
}

// TestValidateV1PromptFormat verifies the locked v1 section check.
func TestValidateV1PromptFormat(t *testing.T) {
	full := `{"episode":{},"scenes":[{"shots":[{"id":"A","prompt":{"en":"Scene & Mood:\n\nFrame Map:\n\nCapture Realism: matte\n\nCamera Capture: handheld"}}]}]}`
	missing := `{"episode":{},"scenes":[{"shots":[{"id":"A","prompt":{"en":"Scene & Mood:\n\nFrame Map:"}}]}]}`
	if !validateV1PromptFormat(full) {
		t.Error("prompt.en with both sections should pass validateV1PromptFormat")
	}
	if validateV1PromptFormat(missing) {
		t.Error("prompt.en without the sections should fail validateV1PromptFormat")
	}
}

// TestJSONExampleIsValid checks that the JSON structure in the prompt is valid.
func TestJSONExampleIsValid(t *testing.T) {
	example := `{"episode":{"title":"Test","totalDuration":120},"scenes":[{"scriptNumber":56,"scriptLocation":"INT. KITCHEN — DAY","title":"Test","description":"Test","duration":25,"start":0,"end":25,"sceneType":"present","mode":"M1","continuity":{"location":"INT. KITCHEN — DAY","locationChange":false,"timeContinuity":"DAY","charactersPresent":["Wyatt"]},"references":[],"shots":[{"id":"A","title":"Test","description":"Test","duration":10,"start":0,"end":10,"camera":{"lens":"40mm","framing":"Medium","movement":"Handheld","fps":24,"shutter":"180 degree","aspectRatio":"9:16"},"composition":{"frameMap":"x=50%","subjectLock":"none","crossFrameRules":"none","focus":"subject","depth":"Shallow DOF"},"blocking":{"location":"Kitchen","movement":"None","interaction":"None","positions":[]},"acting":{"emotion":"Calm","bodyLanguage":"Still","dialogue":"none — silent","microExpressions":[]},"timeline":{"duration":10,"segments":[],"beats":[]},"audio":{"dialogue":"","ambient":"Room tone","sfx":[],"music":false},"references":[],"prompt":{"en":"Scene & Mood:\n\nFrame Map:"},"render":{"mode":"M1","engine":"Seedance"},"notes":{"todos":[],"warnings":[],"approved":false}}]}]}`

	var parsed interface{}
	if err := json.Unmarshal([]byte(example), &parsed); err != nil {
		t.Fatalf("JSON example does not parse: %v", err)
	}
	if !validateShotJSON(example) {
		t.Error("example JSON should pass validateShotJSON")
	}
}

// TestPreFillWorks verifies pre-fill "{" produces valid JSON when
// concatenated with Claude's continuation.
func TestPreFillWorks(t *testing.T) {
	claudeContinuation := `"episode":{"title":"EP16","totalDuration":120},"scenes":[{"scriptNumber":56,"scriptLocation":"INT. WYATT'S KITCHEN","title":"Wyatt confronts Dixie","duration":25,"shots":[{"id":"A","title":"Test","duration":10}]}]}`

	fullJSON := "{" + claudeContinuation
	if !validateShotJSON(fullJSON) {
		t.Error("pre-filled response should validate as JSON")
	}
}

// TestBuildCorrectivePrompt verifies the retry prompt resends the ORIGINAL
// script (guards a documented bug: a retry without originalPrompt made Claude
// answer "No script provided").
func TestBuildCorrectivePrompt(t *testing.T) {
	original := "56. INT. WYATT'S KITCHEN — DAY\nContent here."
	prompt := buildCorrectivePrompt(original)

	if !strings.Contains(prompt, original) {
		t.Error("corrective prompt must resend the ORIGINAL script")
	}
	for _, s := range []string{"ORIGINAL SCRIPT", "not valid JSON", "Regenerate the full shot breakdown"} {
		if !strings.Contains(prompt, s) {
			t.Errorf("corrective prompt missing required part: %q", s)
		}
	}
}

// TestBuildExhaustionError verifies the final error message after all retries
// report the attempt count and the extracted JSON of the last reply.
func TestBuildExhaustionError(t *testing.T) {
	msg := buildExhaustionError(3, `Here: {"episode":{},"scenes":[]}`)
	if !strings.Contains(msg, "after 3 attempts") {
		t.Errorf("message must report the attempt count: %q", msg)
	}
	if !strings.Contains(msg, `{"episode":{},"scenes":[]}`) {
		t.Errorf("message must contain the extracted last response: %q", msg)
	}
}

// TestParseOptimizeResponse verifies the Proncer response parser.
func TestParseOptimizeResponse(t *testing.T) {
	tests := []struct {
		input       string
		expectOptim bool
		desc        string
	}{
		{
			input:       `{"optimized_prompt":"Better prompt","suggestions":["Add light"],"changes_made":["Fixed"]}`,
			expectOptim: true,
			desc:        "clean JSON",
		},
		{
			input:       `Here: {"optimized_prompt":"Better prompt","suggestions":[],"changes_made":[]}`,
			expectOptim: true,
			desc:        "text before JSON",
		},
		{
			input:       `not json`,
			expectOptim: false,
			desc:        "no JSON — returns input as-is",
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			optim, _, _ := parseOptimizeResponse(tc.input)
			if tc.expectOptim && optim == "" {
				t.Error("expected an optimized prompt but got empty")
			}
			if !tc.expectOptim && optim != tc.input {
				t.Errorf("expected input as-is, got %q", optim)
			}
		})
	}
}

// TestExtractJSONStringBraces verifies brace matching inside JSON strings
// does not interfere with extraction.
func TestExtractJSONStringBraces(t *testing.T) {
	input := `Some text: {"episode":{},"scenes":[{"shots":[{"id":"A","en":"Scene & Mood: {mood here}\n\nFrame Map: {map here} {another}"}]}]} trailing`

	expected := `{"episode":{},"scenes":[{"shots":[{"id":"A","en":"Scene & Mood: {mood here}\n\nFrame Map: {map here} {another}"}]}]}`

	got := extractJSON(input)
	if got != expected {
		t.Errorf("\n  got:  %s\n  want: %s", got[:min(len(got), 100)], expected[:min(len(expected), 100)])
	}
	// Validate it actually parses
	if !validateShotJSON(got) {
		t.Error("extracted JSON should validate")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestBuildCorrectivePromptFrom verifies the refine retry prompt resends the
// previous breakdown + change request (never the truncated previous reply).
func TestBuildCorrectivePromptFrom(t *testing.T) {
	previous := `{"episode":{"title":"EP16"},"scenes":[]}`
	original := "=== Previous Breakdown ===\n" + previous + "\n\n=== Change Request ===\nMake it darker"
	prompt := buildCorrectivePromptFrom(original, "=== PREVIOUS BREAKDOWN AND CHANGE REQUEST ===")

	if !strings.Contains(prompt, previous) {
		t.Error("corrective prompt must resend the previous breakdown")
	}
	for _, s := range []string{"PREVIOUS BREAKDOWN", "not valid JSON", "Regenerate the full shot breakdown"} {
		if !strings.Contains(prompt, s) {
			t.Errorf("corrective prompt missing required part: %q", s)
		}
	}
}

// TestRefineModeInstructions verifies the refinement rules include the
// anti-drift directives that keep unchanged shots identical.
func TestRefineModeInstructions(t *testing.T) {
	for _, s := range []string{"change_request", "IDENTICAL", "previous breakdown", "ONLY a valid JSON object", "TARGETED SHOTS", "RECENT CONVERSATION"} {
		if !strings.Contains(refineModeInstructions, s) {
			t.Errorf("refine mode instructions missing required keyword: %q", s)
		}
	}
}

func TestBuildRefinePrompt(t *testing.T) {
	const prev = `{"scenes":[{"scriptNumber":89}]}`
	const change = "make shot A slower"

	t.Run("base compose", func(t *testing.T) {
		p := buildRefinePrompt(nil, prev, change, nil, nil)
		if !strings.Contains(p, "=== Previous Breakdown ===") || !strings.Contains(p, prev) {
			t.Error("missing previous breakdown section")
		}
		if !strings.Contains(p, "=== Change Request ===") || !strings.Contains(p, change) {
			t.Error("missing change request section")
		}
		if strings.Contains(p, "TARGETED SHOTS") || strings.Contains(p, "RECENT CONVERSATION") {
			t.Error("optional sections should be absent when not provided")
		}
	})

	t.Run("targets listed with scene-shot ids", func(t *testing.T) {
		p := buildRefinePrompt(nil, prev, change, []ShotRefineTarget{{SceneNumber: 89, ShotID: "A"}}, nil)
		if !strings.Contains(p, "=== TARGETED SHOTS ===") || !strings.Contains(p, "89-A") {
			t.Error("targeted shots section should list 89-A")
		}
		if !strings.Contains(p, "ONLY to these shots") {
			t.Error("targeted section should scope the change to the listed shots")
		}
	})

	t.Run("recent context bounded and truncated", func(t *testing.T) {
		long := strings.Repeat("x", 800)
		p := buildRefinePrompt(nil, prev, change, nil, []ChatTurn{{Role: "user", Content: long}})
		if !strings.Contains(p, "=== RECENT CONVERSATION ===") {
			t.Error("recent conversation section missing")
		}
		if !strings.Contains(p, "user: ") {
			t.Error("recent turn should be labeled by role")
		}
		if strings.Contains(p, strings.Repeat("x", 700)) {
			t.Error("recent content should be truncated to ~500 runes")
		}
		if !strings.Contains(p, "…") {
			t.Error("truncated content should end with an ellipsis")
		}
	})

	t.Run("scene context included when provided", func(t *testing.T) {
		ctx := &SceneContext{Description: "office"}
		p := buildRefinePrompt(ctx, prev, change, nil, nil)
		if !strings.Contains(p, "=== Current Scene Context ===") || !strings.Contains(p, "office") {
			t.Error("scene context section should be included when provided")
		}
	})

	t.Run("truncateRune leaves short strings intact", func(t *testing.T) {
		if got := truncateRune("hola", 10); got != "hola" {
			t.Errorf("truncateRune should not cut short strings, got %q", got)
		}
	})
}
