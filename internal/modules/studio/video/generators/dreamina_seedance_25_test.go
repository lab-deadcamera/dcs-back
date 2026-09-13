package generators

import (
	"testing"

	"dcs-back-v0/internal/modules/studio"
)

func TestSeedance25Match(t *testing.T) {
	g := &Seedance25Generator{}
	if !g.Match("dreamina-seedance-2-5-260628") {
		t.Error("expected match for dreamina-seedance-2-5-260628")
	}
	if !g.Match("Dreamina-Seedance-2-5-260628-fast") {
		t.Error("expected partial case-insensitive match")
	}
	if g.Match("dreamina-seedance-2-0-260128") {
		t.Error("did not expect match for 2.0 model")
	}
	if g.Match("dreamina-seedance-2-0-gallery") {
		t.Error("did not expect match for gallery model")
	}
}

func TestSeedance25Validate_Valid(t *testing.T) {
	g := &Seedance25Generator{}
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-5-260628",
		Duration: 10,
		Ratio:    "16:9",
		Content: []studio.ContentItem{
			{Type: "text", Text: "a dog walking on the beach"},
		},
	}
	if err := g.Validate(req); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSeedance25Validate_DurationBounds(t *testing.T) {
	g := &Seedance25Generator{}

	// Seedance 2.5 supports 4-30s: below 4 and above 30 are invalid,
	// unlike the 2.0 generator which accepts 1-60.
	for _, tc := range []struct {
		duration int
		wantErr  bool
	}{
		{3, true},
		{4, false},
		{30, false},
		{31, true},
	} {
		req := &studio.GeneratorRequest{
			Model:    "dreamina-seedance-2-5-260628",
			Duration: tc.duration,
			Content: []studio.ContentItem{
				{Type: "text", Text: "a prompt"},
			},
		}
		err := g.Validate(req)
		if tc.wantErr && err == nil {
			t.Errorf("duration %d: expected error, got nil", tc.duration)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("duration %d: expected no error, got: %v", tc.duration, err)
		}
	}
}

func TestSeedance25Validate_InvalidRatio(t *testing.T) {
	g := &Seedance25Generator{}
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-5-260628",
		Duration: 10,
		Ratio:    "invalid",
		Content: []studio.ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error for invalid ratio")
	}
}

func TestSeedance25BuildPayload_Model(t *testing.T) {
	g := &Seedance25Generator{}
	req := &studio.GeneratorRequest{
		Model:      "dreamina-seedance-2-5-260628",
		Duration:   10,
		Ratio:      "9:16",
		Resolution: "720p",
		Content: []studio.ContentItem{
			{Type: "text", Text: "a cinematic scene"},
		},
	}
	payload := g.BuildPayload(req)

	if payload["model"] != nameModelDreaminaSeedance25 {
		t.Errorf("model = %v, want %s", payload["model"], nameModelDreaminaSeedance25)
	}
	if payload["duration"] != 10 {
		t.Errorf("duration = %v, want 10", payload["duration"])
	}
	if payload["ratio"] != "9:16" {
		t.Errorf("ratio = %v, want 9:16", payload["ratio"])
	}
	if payload["resolution"] != "720p" {
		t.Errorf("resolution = %v, want 720p", payload["resolution"])
	}
}

func TestSeedance25BuildPayload_WithReferences(t *testing.T) {
	g := &Seedance25Generator{}
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-5-260628",
		Duration: 10,
		Content: []studio.ContentItem{
			{Type: "text", Text: "a scene"},
			{Type: "image", DataURL: "https://example.com/img.png", ID: "img-uuid"},
			{Type: "video", DataURL: "https://example.com/vid.mp4", ID: "vid-uuid"},
			{Type: "audio", DataURL: "https://example.com/aud.mp3", ID: "aud-uuid"},
		},
	}
	payload := g.BuildPayload(req)

	content, ok := payload["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("content is not []map[string]interface{}")
	}
	if len(content) != 4 {
		t.Fatalf("expected 4 content items (text, image, video, audio), got %d", len(content))
	}

	types := map[string]bool{}
	for _, item := range content {
		types[item["type"].(string)] = true
	}
	for _, want := range []string{"text", "image_url", "video_url", "audio_url"} {
		if !types[want] {
			t.Errorf("expected %s in content", want)
		}
	}
}

func TestSeedance25BuildPayload_EmptyImageDataURLSkipped(t *testing.T) {
	g := &Seedance25Generator{}
	req := &studio.GeneratorRequest{
		Model:    "dreamina-seedance-2-5-260628",
		Duration: 10,
		Content: []studio.ContentItem{
			{Type: "text", Text: "a prompt"},
			{Type: "image", DataURL: "", ID: "uuid"},
		},
	}
	payload := g.BuildPayload(req)

	content := payload["content"].([]map[string]interface{})
	if len(content) != 1 {
		t.Fatalf("expected 1 content item (empty image skipped), got %d", len(content))
	}
	if content[0]["type"] != "text" {
		t.Errorf("content[0].type = %v, want text", content[0]["type"])
	}
}
