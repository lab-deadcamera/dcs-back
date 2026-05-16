package generators

import (
	"testing"
)

func TestSeedreamValidate_Valid(t *testing.T) {
	g := &SeedreamGenerator{}
	req := &GeneratorRequest{
		Model: "dreamina-seedream-4-pro-251224",
		Content: []ContentItem{
			{Type: "text", Text: "a cyberpunk city at night"},
		},
	}
	if err := g.Validate(req); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSeedreamValidate_ValidWithImageRef(t *testing.T) {
	g := &SeedreamGenerator{}
	req := &GeneratorRequest{
		Model: "dreamina-seedream-4-pro-251224",
		Content: []ContentItem{
			{Type: "text", Text: "a cyberpunk city at night"},
			{Type: "image", Text: "reference style", DataURL: "data:image/png;base64,abc", ID: "uuid"},
		},
	}
	if err := g.Validate(req); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSeedreamValidate_DurationNotSupported(t *testing.T) {
	g := &SeedreamGenerator{}
	req := &GeneratorRequest{
		Model:    "dreamina-seedream-4-pro-251224",
		Duration: 5,
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error: duration not supported for image generation")
	}
}

func TestSeedreamValidate_CameraFixedNotSupported(t *testing.T) {
	g := &SeedreamGenerator{}
	req := &GeneratorRequest{
		Model:       "dreamina-seedream-4-pro-251224",
		CameraFixed: true,
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error: camerafixed not supported for image generation")
	}
}

func TestSeedreamValidate_AudioNotSupported(t *testing.T) {
	g := &SeedreamGenerator{}
	req := &GeneratorRequest{
		Model:         "dreamina-seedream-4-pro-251224",
		GenerateAudio: true,
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error: generate_audio not supported for image generation")
	}
}

func TestSeedreamValidate_InvalidResolution(t *testing.T) {
	g := &SeedreamGenerator{}
	req := &GeneratorRequest{
		Model:      "dreamina-seedream-4-pro-251224",
		Resolution: "480p",
		Content: []ContentItem{
			{Type: "text", Text: "a prompt"},
		},
	}
	if err := g.Validate(req); err == nil {
		t.Fatal("expected error: invalid resolution for image model")
	}
}
