package text

import "testing"

// ─── resolveSystemPrompt (used by shot builder) — fallback paths ──────
//
// The skill resolution path (skillID → DB lookup) simply delegates to
// skillSvc.GetByID which is tested in the skill module itself.
// These tests focus on the fallback decisions that differ between
// resolveSystemPrompt and resolveSystemPromptStrict.

func TestResolveSystemPrompt_NoSkillNoRequest_UsesDefault(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPrompt("", "")
	if got != defaultShotBuilderPrompt {
		t.Errorf("expected defaultShotBuilderPrompt, got %q (len=%d)", got[:40]+"...", len(got))
	}
}

func TestResolveSystemPrompt_RequestPromptGiven_ReturnsIt(t *testing.T) {
	h := &Handler{skillSvc: nil}
	prompt := "custom-system-prompt"
	got := h.resolveSystemPrompt("", prompt)
	if got != prompt {
		t.Errorf("expected %q, got %q", prompt, got)
	}
}

func TestResolveSystemPrompt_SkillIDButNoSkillService_FallsBackToRequestPrompt(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPrompt("any-skill", "req-prompt")
	if got != "req-prompt" {
		t.Errorf("expected request prompt, got %q", got)
	}
}

func TestResolveSystemPrompt_SkillIDButNoSkillServiceNoRequest_UsesDefault(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPrompt("any-skill", "")
	if got != defaultShotBuilderPrompt {
		t.Errorf("expected defaultShotBuilderPrompt, got %q", got[:40]+"...")
	}
}

// ─── resolveSystemPromptStrict (used by proncer) — fallback paths ─────

func TestResolveSystemPromptStrict_NoSkillNoRequest_ReturnsEmpty(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPromptStrict("", "")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestResolveSystemPromptStrict_RequestPromptGiven_ReturnsIt(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPromptStrict("", "proncer-prompt")
	if got != "proncer-prompt" {
		t.Errorf("expected %q, got %q", "proncer-prompt", got)
	}
}

func TestResolveSystemPromptStrict_SkillIDButNoSkillService_FallsBackToRequestPrompt(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPromptStrict("missing", "req-prompt")
	if got != "req-prompt" {
		t.Errorf("expected request prompt, got %q", got)
	}
}

func TestResolveSystemPromptStrict_SkillIDButNoSkillServiceNoRequest_ReturnsEmpty(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPromptStrict("missing", "")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ─── Cross-contamination guard ────────────────────────────────────────
//
// These tests guarantee the two resolution functions never return each
// other's default — the most common class of bug when one endpoint
// accidentally receives the other's system prompt.

func TestCrossContamination_StrictNeverLeaksShotBuilderDefault(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPromptStrict("", "")
	if got == defaultShotBuilderPrompt {
		t.Error("strict mode must NOT return defaultShotBuilderPrompt")
	}
}

func TestCrossContamination_ShotBuilderNeverReturnsEmptyWhenNoConfig(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPrompt("", "")
	if got == "" {
		t.Error("shot builder must fall back to a default prompt, not empty")
	}
}

// ─── Request prompt overrides skillID when skill service is missing ──

func TestCrossContamination_ShotBuilderRequestPromptTakesPriorityOverUnknownSkill(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPrompt("some-skill", "user-prompt")
	if got != "user-prompt" {
		t.Errorf("expected request prompt when skill service is nil, got %q", got)
	}
}

func TestCrossContamination_StrictRequestPromptTakesPriorityOverUnknownSkill(t *testing.T) {
	h := &Handler{skillSvc: nil}
	got := h.resolveSystemPromptStrict("some-skill", "user-prompt")
	if got != "user-prompt" {
		t.Errorf("expected request prompt when skill service is nil, got %q", got)
	}
}
