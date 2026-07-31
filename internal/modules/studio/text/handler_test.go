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
		"Scene and Mood",   // simplified pre-prompt format
		"Composition",
		"Space and Mélange",
		"Cross-Shot Rule",
		"Action",
		"Dialogue",
		"Ending Shot",
		"Environmental Base",
		"Sound Layer",
		"Severe shaking, time flickering, and identity drift were avoided",
		"micro-fidgeting",  // acting features
		"Anatomical Emotion",
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
