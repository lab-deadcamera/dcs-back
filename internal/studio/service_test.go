package studio

import (
	"testing"
	"time"

	
)

func TestDetectAssetType(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"video/mp4", "Video"},
		{"video/quicktime", "Video"},
		{"audio/mpeg", "Audio"},
		{"audio/wav", "Audio"},
		{"image/png", "Image"},
		{"image/jpeg", "Image"},
		{"application/octet-stream", "Image"},
		{"", "Image"},
	}
	for _, tc := range tests {
		got := detectAssetType(tc.mime)
		if got != tc.want {
			t.Errorf("detectAssetType(%q) = %q, want %q", tc.mime, got, tc.want)
		}
	}
}

func TestConvertOutputs_Nil(t *testing.T) {
	result := convertOutputs(nil)
	if result == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d", len(result))
	}
}

func TestConvertOutputs_Empty(t *testing.T) {
	result := convertOutputs([]OutputResource{})
	if result == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d", len(result))
	}
}

func TestConvertOutputs_WithResources(t *testing.T) {
	src := []OutputResource{
		{URL: "https://example.com/vid.mp4", LocalURL: "/outputs/vid.mp4", Type: "video"},
		{URL: "https://example.com/img.png", Type: "image"},
	}
	result := convertOutputs(src)
	if len(result) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(result))
	}
	if result[0].URL != "https://example.com/vid.mp4" {
		t.Errorf("outputs[0].URL = %q", result[0].URL)
	}
	if result[0].LocalURL != "/outputs/vid.mp4" {
		t.Errorf("outputs[0].LocalURL = %q", result[0].LocalURL)
	}
	if result[0].Type != "video" {
		t.Errorf("outputs[0].Type = %q", result[0].Type)
	}
	if result[1].URL != "https://example.com/img.png" {
		t.Errorf("outputs[1].URL = %q", result[1].URL)
	}
	if result[1].LocalURL != "" {
		t.Errorf("expected empty LocalURL, got %q", result[1].LocalURL)
	}
	if result[1].Type != "image" {
		t.Errorf("outputs[1].Type = %q", result[1].Type)
	}
}

func TestListGenerationLogsResponse_PageCalculation(t *testing.T) {
	tests := []struct {
		total      int
		limit      int
		wantPages int
	}{
		{0, 20, 0},
		{1, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{40, 20, 2},
		{41, 20, 3},
	}
	for _, tc := range tests {
		pages := (tc.total + tc.limit - 1) / tc.limit
		if pages != tc.wantPages {
			t.Errorf("total=%d limit=%d: got %d pages, want %d", tc.total, tc.limit, pages, tc.wantPages)
		}
	}
}

func TestGenerationLog_Timestamps(t *testing.T) {
	now := time.Now()
	log := GenerationLog{
		ID:        "uuid",
		TaskID:    "task-123",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if log.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if log.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
	if log.DeletedAt != nil {
		t.Error("DeletedAt should be nil by default")
	}
}

func TestGenerationLog_SoftDelete(t *testing.T) {
	now := time.Now()
	log := GenerationLog{
		ID:        "uuid",
		TaskID:    "task-123",
		DeletedAt: &now,
	}
	if log.DeletedAt == nil {
		t.Error("DeletedAt should not be nil")
	}
}
