package generators

import (
	"testing"

	"dcs-back-v0/internal/studio"
)

func TestSeedanceValidate_Valid(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Ratio:    "16:9",
		Content: []studio.ContentItem{
			{Type: "text", Text: "a dog walking on the beach"},
		},
	}
	if err := g.Validate(req); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSeedanceValidate_InvalidDuration(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 0,
		Content: []studio.ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error for duration < 1")
	}
}

func TestSeedanceValidate_InvalidRatio(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Ratio:    "invalid",
		Content: []studio.ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error for invalid ratio")
	}
}

func TestSeedanceValidate_InvalidResolution(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &studio.GeneratorRequest{
		Model:      "dreamina-seedance-2-0-260128",
		Duration:   5,
		Resolution: "4K",
		Content: []studio.ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error for invalid resolution")
	}
}

func TestSeedanceValidate_AudioOnFastModel(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &studio.GeneratorRequest{
		Model:         "dreamina-seedance-2-0-fast-260128",
		Duration:      5,
		GenerateAudio: true,
		Content: []studio.ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error: audio not supported on fast models")
	}
}

func TestSeedanceBuildPayload_TextOnly(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &studio.GeneratorRequest{
		Model:       "dreamina-seedance-2-0-fast-260128",
		Duration:    5,
		Ratio:       "16:9",
		CameraFixed: true,
		Watermark:   false,
		Content: []studio.ContentItem{
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
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Content: []studio.ContentItem{
			{Type: "text", Text: "a dog on the beach"},
			{Type: "image", Text: "", DataURL: "https://example.com/img123.png", ID: "uuid"},
		},
	}
	payload := g.BuildPayload(req)

	content, ok := payload["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("content is not []map[string]interface{}")
	}

	if len(content) != 2 {
		t.Fatalf("expected 2 content items (image_url, text), got %d", len(content))
	}

	if content[0]["type"] != "image_url" {
		t.Errorf("content[0].type = %v, want image_url", content[0]["type"])
	}
	imageURL, ok := content[0]["image_url"].(map[string]string)
	if !ok || imageURL["url"] != "https://example.com/img123.png" {
		t.Errorf("content[0].image_url.url = %v, want https://example.com/img123.png", imageURL)
	}

	if content[1]["type"] != "text" {
		t.Errorf("content[1].type = %v, want text", content[1]["type"])
	}
}

func TestSeedanceBuildPayload_WithMultipleImages(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Content: []studio.ContentItem{
			{Type: "text", Text: "a prompt"},
			{Type: "image", DataURL: "https://example.com/img1.png", ID: "u1"},
			{Type: "image", DataURL: "https://example.com/img2.png", ID: "u2"},
		},
	}
	payload := g.BuildPayload(req)

	content := payload["content"].([]map[string]interface{})

	if len(content) != 3 {
		t.Fatalf("expected 3 content items, got %d", len(content))
	}

	imageCount := 0
	for _, item := range content {
		if item["type"] == "image_url" {
			imageCount++
		}
	}
	if imageCount != 2 {
		t.Errorf("expected 2 image_url items, got %d", imageCount)
	}
}

func TestSeedanceBuildPayload_EmptyImageDataURL(t *testing.T) {
	g := &SeedanceGenerator{}
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Content: []studio.ContentItem{
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
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-0-260128",
		Duration: 5,
		Content: []studio.ContentItem{
			{Type: "text", Text: "a cinematic scene"},
			{Type: "video", DataURL: "https://example.com/vid1.mp4", ID: "vid-uuid"},
			{Type: "audio", DataURL: "https://example.com/aud1.mp3", ID: "aud-uuid"},
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
		name     string
		model    string
		audioReq bool
		want     bool
	}{
		{"pro model with audio true", "dreamina-seedance-2-0-260128", true, true},
		{"pro model with audio false", "dreamina-seedance-2-0-260128", false, false},
		{"fast model with audio true", "dreamina-seedance-2-0-fast-260128", true, true},
		{"fast model with audio false", "dreamina-seedance-2-0-fast-260128", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &studio.GeneratorRequest{
				Model:         tc.model,
				Duration:      5,
				GenerateAudio: tc.audioReq,
				Content: []studio.ContentItem{
					{Type: "text", Text: "a prompt"},
				},
			}
			g := &SeedanceGenerator{}
			payload := g.BuildPayload(req)

			val, has := payload["generate_audio"]
			if !has {
				t.Error("expected generate_audio in payload")
			}
			b, ok := val.(bool)
			if !ok {
				t.Errorf("generate_audio should be bool, got %T", val)
			}
			if b != tc.want {
				t.Errorf("generate_audio = %v, want %v", b, tc.want)
			}
		})
	}
}
