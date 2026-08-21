package text

import (
	"strings"
	"testing"
)

// TestElementElicitationPromptNoBackticks verifies the prompt constant has no
// raw backticks that would break Go string syntax.
func TestElementElicitationPromptNoBackticks(t *testing.T) {
	if strings.Contains(elementElicitationPrompt, "`") {
		t.Error("elementElicitationPrompt contains raw backtick character")
	}
}

// TestElementElicitationPromptHasFormatRule verifies the CRITICAL output format
// rule is present.
func TestElementElicitationPromptHasFormatRule(t *testing.T) {
	if !strings.Contains(elementElicitationPrompt, "CRITICAL — Output Format") {
		t.Error("prompt missing CRITICAL output format rule")
	}
	if !strings.Contains(elementElicitationPrompt, "valid JSON object") {
		t.Error("prompt missing 'valid JSON object' instruction")
	}
}

// TestElementElicitationPromptMentionsSchema verifies the prompt documents the
// full wire-format contract (snake_case fields, categories and statuses).
func TestElementElicitationPromptMentionsSchema(t *testing.T) {
	required := []string{
		"element_registry",
		"entity_id",
		"mentioned_as",
		"source_text",
		"scene_number",
		"definition_status",
		"consistency_group",
		"linked_asset_id",
		"defined",
		"asset_orphan",
		"undefined",
		"character",
		"prop",
		"location",
		"vehicle",
		"weather",
		"wardrobe",
		"screen_content",
		"sound_object",
	}
	for _, s := range required {
		if !strings.Contains(elementElicitationPrompt, s) {
			t.Errorf("prompt missing required field/keyword: %q", s)
		}
	}
}

// TestValidateElementRegistryJSON covers the analyze-elements response
// validation: happy path plus every rejection branch.
func TestValidateElementRegistryJSON(t *testing.T) {
	valid := `{"element_registry":[{"entity_id":"e1","mentioned_as":"the red door","source_text":"he knocks on the red door","scene_number":2,"category":"prop","definition_status":"undefined","consistency_group":"red_door"}],"summary":"1 gap"}`
	if err := validateElementRegistryJSON(valid); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}

	tests := []struct {
		name  string
		input string
	}{
		{"not json", "this is not json at all"},
		{"empty registry", `{"element_registry":[],"summary":"none"}`},
		{"missing registry", `{"summary":"none"}`},
		{"missing entity_id", `{"element_registry":[{"mentioned_as":"door","scene_number":1,"category":"prop","definition_status":"undefined","consistency_group":"d"}]}`},
		{"missing consistency_group", `{"element_registry":[{"entity_id":"e1","mentioned_as":"door","scene_number":1,"category":"prop","definition_status":"undefined"}]}`},
		{"invalid category", `{"element_registry":[{"entity_id":"e1","mentioned_as":"door","scene_number":1,"category":"spaceship","definition_status":"undefined","consistency_group":"d"}]}`},
		{"invalid status", `{"element_registry":[{"entity_id":"e1","mentioned_as":"door","scene_number":1,"category":"prop","definition_status":"pending","consistency_group":"d"}]}`},
		{"negative scene_number", `{"element_registry":[{"entity_id":"e1","mentioned_as":"door","scene_number":-1,"category":"prop","definition_status":"undefined","consistency_group":"d"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateElementRegistryJSON(tt.input); err == nil {
				t.Errorf("expected error for %s payload, got nil", tt.name)
			}
		})
	}
}

// TestBuildClosedWorldBlock verifies each decision type renders its binding
// directive and undecided entries fall back to their analysis status.
func TestBuildClosedWorldBlock(t *testing.T) {
	registry := []ElementEntity{
		{EntityID: "e1", MentionedAs: "the red door", SceneNumber: 2, Category: "prop", DefinitionStatus: "asset_orphan", ConsistencyGroup: "red_door", LinkedAssetID: "file-42",
			UserDecision: &ElementDecision{Type: "define_with_reference"}},
		{EntityID: "e2", MentionedAs: "a brass lantern", SceneNumber: 3, Category: "prop", DefinitionStatus: "undefined", ConsistencyGroup: "lantern",
			UserDecision: &ElementDecision{Type: "define_with_text", Description: "oxidized brass, cracked glass pane"}},
		{EntityID: "e3", MentionedAs: "the dog", SceneNumber: 3, Category: "animal", DefinitionStatus: "undefined", ConsistencyGroup: "dog",
			UserDecision: &ElementDecision{Type: "invent_restricted", Description: "small, brown, no collar"}},
		{EntityID: "e4", MentionedAs: "fog", SceneNumber: 4, Category: "weather", DefinitionStatus: "defined", ConsistencyGroup: "fog"},
	}

	block := buildClosedWorldBlock(registry)

	required := []string{
		"## Closed-World Mode (element registry)",
		"HARD rules",
		`[prop] "the red door" (scene 2): USE the linked asset reference EXACTLY as provided`,
		`[prop] "a brass lantern" (scene 3): render EXACTLY as described — oxidized brass, cracked glass pane`,
		`[animal] "the dog" (scene 3): invent WITHIN these constraints — small, brown, no collar`,
		`[weather] "fog" (scene 4): status defined`,
		"Do NOT introduce new named visual elements beyond this registry",
	}
	for _, s := range required {
		if !strings.Contains(block, s) {
			t.Errorf("closed-world block missing %q\nblock:\n%s", s, block)
		}
	}
}

// TestBuildClosedWorldBlockEmpty verifies an empty registry still produces a
// well-formed block (callers guard on len>0, but the helper must not panic).
func TestBuildClosedWorldBlockEmpty(t *testing.T) {
	block := buildClosedWorldBlock(nil)
	if !strings.Contains(block, "## Closed-World Mode (element registry)") {
		t.Errorf("empty registry produced malformed block: %q", block)
	}
}
