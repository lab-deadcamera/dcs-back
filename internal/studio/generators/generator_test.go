package generators

import (
	"testing"
)

func TestValidateCommon_Valid(t *testing.T) {
	req := &GeneratorRequest{
		Model: "dreamina-seedance-2-0-260128",
		Content: []ContentItem{
			{Type: "text", Text: "a dog walking on the beach"},
		},
	}
	errs := validateCommon(req)
	if errs.HasErrors() {
		t.Errorf("expected no errors, got: %v", errs.Error())
	}
}

func TestValidateCommon_MissingModel(t *testing.T) {
	req := &GeneratorRequest{
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	errs := validateCommon(req)
	if !errs.HasErrors() {
		t.Fatal("expected errors")
	}
	if len(errs.Fields) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs.Fields), errs.Fields)
	}
}

func TestValidateCommon_MissingContent(t *testing.T) {
	req := &GeneratorRequest{
		Model:   "some-model",
		Content: []ContentItem{},
	}
	errs := validateCommon(req)
	if !errs.HasErrors() {
		t.Fatal("expected errors")
	}
}

func TestValidateCommon_ContentWithoutText(t *testing.T) {
	req := &GeneratorRequest{
		Model: "some-model",
		Content: []ContentItem{
			{Type: "image", Text: "", ID: "uuid", DataURL: "data:image/png;base64,abc"},
		},
	}
	errs := validateCommon(req)
	if !errs.HasErrors() {
		t.Fatal("expected errors — no text item with a prompt")
	}
}

func TestValidateCommon_MissingContentType(t *testing.T) {
	req := &GeneratorRequest{
		Model: "some-model",
		Content: []ContentItem{
			{Type: "", Text: ""},
		},
	}
	errs := validateCommon(req)
	if !errs.HasErrors() {
		t.Fatal("expected errors")
	}
}

func TestValidateCommon_MultipleErrors(t *testing.T) {
	req := &GeneratorRequest{
		Model:   "",
		Content: []ContentItem{},
	}
	errs := validateCommon(req)
	if !errs.HasErrors() {
		t.Fatal("expected errors")
	}
	if len(errs.Fields) < 2 {
		t.Fatalf("expected at least 2 errors (model + content), got %d: %v", len(errs.Fields), errs.Fields)
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Fields: []string{"model: is required", "content: must have at least one item"},
	}
	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestIsFastModel(t *testing.T) {
	tests := []struct {
		model string
		fast  bool
	}{
		{"dreamina-seedance-2-0-fast-260128", true},
		{"dreamina-seedance-2-0-260128", false},
		{"DREAMINA-SEEDANCE-2-0-FAST-260128", true},
		{"", false},
	}
	for _, tc := range tests {
		got := isFastModel(tc.model)
		if got != tc.fast {
			t.Errorf("isFastModel(%q) = %v, want %v", tc.model, got, tc.fast)
		}
	}
}

func TestSafeSuffix(t *testing.T) {
	tests := []struct {
		taskID string
		want   string
	}{
		{"cgt-20260515-abc123def456", "23def456"},
		{"short", "short"},
		{"", ""},
	}
	for _, tc := range tests {
		got := safeSuffix(tc.taskID)
		if got != tc.want {
			t.Errorf("safeSuffix(%q) = %q, want %q", tc.taskID, got, tc.want)
		}
	}
}

func TestCompileContentText_SingleText(t *testing.T) {
	items := []ContentItem{
		{Type: "text", Text: "a dog walking on the beach"},
	}
	got := compileContentText(items)
	want := "a dog walking on the beach."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCompileContentText_MultipleText(t *testing.T) {
	items := []ContentItem{
		{Type: "text", Text: "first prompt"},
		{Type: "text", Text: "second prompt"},
	}
	got := compileContentText(items)
	want := "first prompt. second prompt."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCompileContentText_WithFileDescriptions(t *testing.T) {
	items := []ContentItem{
		{Type: "text", Text: "main prompt"},
		{Type: "image", Text: "reference image description", DataURL: "data:...", ID: "uuid"},
		{Type: "text", Text: "additional details"},
	}
	got := compileContentText(items)
	// Only text-type items are concatenated, not descriptions from non-text items
	want := "main prompt. additional details."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCompileContentText_Empty(t *testing.T) {
	got := compileContentText([]ContentItem{})
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestCompileContentText_OnlyNonText(t *testing.T) {
	items := []ContentItem{
		{Type: "image", Text: "image desc", DataURL: "data:...", ID: "uuid"},
		{Type: "video", Text: "video desc", DataURL: "data:...", ID: "uuid"},
	}
	got := compileContentText(items)
	// Only text-type items are included, non-text items are ignored
	if got != "" {
		t.Errorf("expected empty string for non-text items only, got %q", got)
	}
}

func TestCompileContentText_EmptyTextItems(t *testing.T) {
	items := []ContentItem{
		{Type: "text", Text: ""},
		{Type: "image", Text: "", DataURL: "data:..."},
	}
	got := compileContentText(items)
	if got != "" {
		t.Errorf("expected empty string for empty text items, got %q", got)
	}
}

func TestCompileContentText_EndsWithPeriod(t *testing.T) {
	tests := []struct {
		input []ContentItem
	}{
		{[]ContentItem{{Type: "text", Text: "hello world"}}},
		{[]ContentItem{{Type: "text", Text: "hello world."}}},
		{[]ContentItem{{Type: "text", Text: "hello world. "}}},
		{[]ContentItem{{Type: "text", Text: "hello"}, {Type: "text", Text: "world"}}},
	}
	for _, tc := range tests {
		result := compileContentText(tc.input)
		if len(result) > 0 && result[len(result)-1] != '.' {
			t.Errorf("result %q should end with a period", result)
		}
	}
}
