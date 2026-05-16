package generators

import (
	"testing"
)

func TestSeedanceValidate_Valid(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Ratio:    "16:9",
		Content: []ContentItem{
			{Type: "text", Text: "a dog walking on the beach"},
		},
	}
	if err := g.Validate(req); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSeedanceValidate_InvalidDuration(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 0,
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error for duration < 1")
	}
}

func TestSeedanceValidate_InvalidRatio(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Ratio:    "invalid",
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error for invalid ratio")
	}
}

func TestSeedanceValidate_InvalidResolution(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:      "dreamina-seedance-2-0-260128",
		Duration:   5,
		Resolution: "4K",
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error for invalid resolution")
	}
}

func TestSeedanceValidate_AudioOnFastModel(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:         "dreamina-seedance-2-0-fast-260128",
		Duration:      5,
		GenerateAudio: true,
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error: audio not supported on fast models")
	}
}

func TestSeedanceBuildPayload_TextOnly(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:       "dreamina-seedance-2-0-fast-260128",
		Duration:    5,
		Ratio:       "16:9",
		CameraFixed: true,
		Watermark:   false,
		Content: []ContentItem{
			{Type: "text", Text: "a dog walking on the beach"},
		},
	}
	payload := g.BuildPayload(req)

	if payload["model"] != req.Model {
		t.Errorf("model = %v, want %s", payload["model"], req.Model)
	}
	if payload["duration"] != req.Duration {
		t.Errorf("duration = %v, want %d", payload["duration"], req.Duration)
	}
	if payload["ratio"] != req.Ratio {
		t.Errorf("ratio = %v, want %s", payload["ratio"], req.Ratio)
	}

	content, ok := payload["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("content is not []map[string]interface{}")
	}
	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}
	if content[0]["type"] != "text" {
		t.Errorf("content[0].type = %v, want text", content[0]["type"])
	}
}

func TestSeedanceBuildPayload_WithImage(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Content: []ContentItem{
			{Type: "text", Text: "a dog on the beach"},
			{Type: "image", Text: "reference", DataURL: "data:image/png;base64,img123", ID: "uuid"},
		},
	}
	payload := g.BuildPayload(req)

	content, ok := payload["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("content is not []map[string]interface{}")
	}

	// Should have: duplicate image ref + image_url + text = 3 items
	if len(content) != 3 {
		t.Fatalf("expected 3 content items (dup image, image_url, text), got %d", len(content))
	}

	// First item should be the duplicated image reference
	if content[0]["type"] != "image_url" {
		t.Errorf("content[0].type = %v, want image_url", content[0]["type"])
	}
	imageURL, ok := content[0]["image_url"].(map[string]string)
	if !ok || imageURL["url"] != "data:image/png;base64,img123" {
		t.Errorf("content[0].image_url.url = %v, want data:image/png;base64,img123", imageURL)
	}
}

func TestSeedanceBuildPayload_FirstImageWithTextFirst(t *testing.T) {
	// Regression test: first content item is text, image is second
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Content: []ContentItem{
			{Type: "text", Text: "a dog on the beach"},
			{Type: "image", Text: "reference", DataURL: "data:image/png;base64,img456", ID: "uuid"},
		},
	}
	payload := g.BuildPayload(req)

	content := payload["content"].([]map[string]interface{})

	// First item = duplicated image ref, should use the image item's DataURL, not text item
	first := content[0]
	imageURL := first["image_url"].(map[string]string)
	if imageURL["url"] != "data:image/png;base64,img456" {
		t.Errorf("first duplicated image url = %q, want data:image/png;base64,img456", imageURL["url"])
	}
}

func TestSeedanceBuildPayload_EmptyImageDataURL(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
			{Type: "image", DataURL: "", ID: "uuid"},
		},
	}
	payload := g.BuildPayload(req)

	content := payload["content"].([]map[string]interface{})
	for _, item := range content {
		if item["type"] == "image_url" {
			img, ok := item["image_url"].(map[string]string)
			if ok && img["url"] == "" {
				t.Error("found image_url with empty url field")
			}
		}
	}
}

func TestSeedanceBuildPayload_WithVideoAndAudio(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Content: []ContentItem{
			{Type: "text", Text: "a cinematic scene"},
			{Type: "video", DataURL: "data:video/mp4;base64,vid1", ID: "vid-uuid"},
			{Type: "audio", DataURL: "data:audio/mpeg;base64,aud1", ID: "aud-uuid"},
		},
	}
	payload := g.BuildPayload(req)

	content := payload["content"].([]map[string]interface{})

	foundVideo := false
	foundAudio := false
	for _, item := range content {
		switch item["type"] {
		case "video_url":
			foundVideo = true
		case "audio_url":
			foundAudio = true
		}
	}
	if !foundVideo {
		t.Error("expected video_url in content")
	}
	if !foundAudio {
		t.Error("expected audio_url in content")
	}
}

func TestSeedanceBuildPayload_GenerateAudio(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{"pro model with audio", "dreamina-seedance-2-0-260128", true},
		{"fast model with audio", "dreamina-seedance-2-0-fast-260128", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &GeneratorRequest{
				Model:         tc.model,
				Duration:      5,
				GenerateAudio: true,
				Content: []ContentItem{
					{Type: "text", Text: "a prompt"},
				},
			}
			g := &SeedanceGenerator{}
			payload := g.BuildPayload(req)

			_, has := payload["generate_audio"]
			if tc.want && !has {
				t.Error("expected generate_audio in payload")
			}
			if !tc.want && has {
				t.Error("did not expect generate_audio in payload for fast model")
			}
		})
	}
}
