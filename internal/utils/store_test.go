package utils

import (
	"os"
	"path/filepath"
	"testing"

	"dcs-back-v0/config"
)

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"video.mp4", MediaTypeVideo},
		{"video.mov", MediaTypeVideo},
		{"video.avi", MediaTypeVideo},
		{"video.mkv", MediaTypeVideo},
		{"video.webm", MediaTypeVideo},
		{"video.flv", MediaTypeVideo},
		{"video.wmv", MediaTypeVideo},
		{"video.m4v", MediaTypeVideo},
		{"video.3gp", MediaTypeVideo},
		{"video.ts", MediaTypeVideo},
		{"audio.mp3", MediaTypeAudio},
		{"audio.wav", MediaTypeAudio},
		{"audio.ogg", MediaTypeAudio},
		{"audio.flac", MediaTypeAudio},
		{"audio.aac", MediaTypeAudio},
		{"audio.m4a", MediaTypeAudio},
		{"audio.wma", MediaTypeAudio},
		{"audio.opus", MediaTypeAudio},
		{"audio.aiff", MediaTypeAudio},
		{"image.jpg", MediaTypeImage},
		{"image.jpeg", MediaTypeImage},
		{"image.png", MediaTypeImage},
		{"image.gif", MediaTypeImage},
		{"image.webp", MediaTypeImage},
		{"image.bmp", MediaTypeImage},
		{"image.svg", MediaTypeImage},
		{"image.tiff", MediaTypeImage},
		{"image.tif", MediaTypeImage},
		{"image.ico", MediaTypeImage},
		{"document.pdf", MediaTypeOther},
		{"archive.zip", MediaTypeOther},
		{"noext", MediaTypeOther},
		{"", MediaTypeOther},
		{"UPPERCASE.MP4", MediaTypeVideo},
	}
	for _, tc := range tests {
		got := DetectMediaType(tc.filename)
		if got != tc.want {
			t.Errorf("DetectMediaType(%q) = %q, want %q", tc.filename, got, tc.want)
		}
	}
}

func TestDecodeBase64(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "plain base64",
			input: "aGVsbG8=",
			want:  "hello",
		},
		{
			name:  "data URI prefix",
			input: "data:image/png;base64,aGVsbG8=",
			want:  "hello",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid base64",
			input:   "not-valid!!!",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeBase64(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("DecodeBase64(%q) expected error", tc.input)
				}
				return
			}
			if err != nil {
				t.Errorf("DecodeBase64(%q) unexpected error: %v", tc.input, err)
				return
			}
			if string(got) != tc.want {
				t.Errorf("DecodeBase64(%q) = %q, want %q", tc.input, string(got), tc.want)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal file", "photo.jpg", "photo.jpg"},
		{"path traversal", "../../etc/passwd", "passwd"},
		{"double dots", "file..txt", "filetxt"},
		{"multiple traversal", "a/../../b", "b"},
		{"whitespace", "  file.txt  ", "file.txt"},
		{"empty", "", "."},
		{"only traversal", "..", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeFilename(tc.input)
			if got != tc.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestDetectMimeFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		data []byte
		want string
	}{
		{
			name: "video extension",
			url:  "https://example.com/video.mp4",
			want: "video/mp4",
		},
		{
			name: "image extension with query",
			url:  "https://example.com/photo.png?w=300",
			want: "image/png",
		},
		{
			name: "audio extension",
			url:  "https://example.com/sound.mp3",
			want: "audio/mpeg",
		},
		{
			name: "unknown extension falls back to content sniff",
			url:  "https://example.com/file.xyz",
			data: []byte("<html></html>"),
			want: "text/html; charset=utf-8",
		},
		{
			name: "no extension",
			url:  "https://example.com/abc123",
			data: []byte{0xFF, 0xD8, 0xFF},
			want: "image/jpeg",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectMimeFromURL([]byte(tc.url), tc.data)
			if got != tc.want {
				t.Errorf("detectMimeFromURL(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}

func TestSaveToOutput(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		filename  string
		wantErr   bool
		wantMatch string
	}{
		{
			name:      "save image",
			data:      []byte("fake-image-data"),
			filename:  "test.png",
			wantMatch: "/outputs/image/test.png",
		},
		{
			name:      "save video",
			data:      []byte("fake-video-data"),
			filename:  "clip.mp4",
			wantMatch: "/outputs/video/clip.mp4",
		},
		{
			name:      "save audio",
			data:      []byte("fake-audio-data"),
			filename:  "song.mp3",
			wantMatch: "/outputs/audio/song.mp3",
		},
		{
			name:      "save other",
			data:      []byte("data"),
			filename:  "doc.pdf",
			wantMatch: "/outputs/other/doc.pdf",
		},
		{
			name:     "empty data",
			data:     []byte{},
			filename: "test.png",
			wantErr:  true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			urlPath, err := SaveToOutput(tc.data, tc.filename)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if urlPath != tc.wantMatch {
				t.Errorf("urlPath = %q, want %q", urlPath, tc.wantMatch)
			}
			mediaType := DetectMediaType(tc.filename)
			expectedFile := filepath.Join(".", config.OutPutUrl(), mediaType, tc.filename)
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Errorf("file not created at %s", expectedFile)
			}
		})
	}
}

func TestSaveToOutput_SanitizesFilename(t *testing.T) {
	urlPath, err := SaveToOutput([]byte("data"), "../../evil.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if urlPath != "/outputs/image/evil.png" {
		t.Errorf("urlPath = %q, want /outputs/image/evil.png", urlPath)
	}
}

func TestSaveOutput_Modes(t *testing.T) {

	t.Run("binary mode", func(t *testing.T) {
		urlPath, err := SaveOutput([]byte("binary-data"), ModeBinary, "file.bin")
		if err != nil {
			t.Fatalf("binary mode error: %v", err)
		}
		if urlPath != "/outputs/other/file.bin" {
			t.Errorf("urlPath = %q", urlPath)
		}
	})

	t.Run("binary mode wrong type", func(t *testing.T) {
		_, err := SaveOutput("not-bytes", ModeBinary, "file.bin")
		if err == nil {
			t.Error("expected error for wrong input type")
		}
	})

	t.Run("base64 mode", func(t *testing.T) {
		urlPath, err := SaveOutput("aGVsbG8=", ModeBase64, "hello.txt")
		if err != nil {
			t.Fatalf("base64 mode error: %v", err)
		}
		if urlPath != "/outputs/other/hello.txt" {
			t.Errorf("urlPath = %q", urlPath)
		}
	})

	t.Run("base64 mode wrong type", func(t *testing.T) {
		_, err := SaveOutput(123, ModeBase64, "file.txt")
		if err == nil {
			t.Error("expected error for wrong input type")
		}
	})

	t.Run("unsupported mode", func(t *testing.T) {
		_, err := SaveOutput(nil, "invalid", "file.txt")
		if err == nil {
			t.Error("expected error for unsupported mode")
		}
	})
}

func TestDetectMimeFromURL_Fallback(t *testing.T) {
	got := detectMimeFromURL([]byte("https://example.com/file"), []byte{0, 0, 0})
	if got == "" {
		t.Error("expected non-empty mime type")
	}
}

func TestSaveBase64Output(t *testing.T) {
	urlPath, err := SaveBase64Output("aGVsbG8=", "test.txt")
	if err != nil {
		t.Fatalf("SaveBase64Output error: %v", err)
	}
	if urlPath != "/outputs/other/test.txt" {
		t.Errorf("urlPath = %q", urlPath)
	}

	_, err = SaveBase64Output("!!!invalid", "test.txt")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestSaveBinaryOutput(t *testing.T) {
	urlPath, err := SaveBinaryOutput([]byte("data"), "file.bin")
	if err != nil {
		t.Fatalf("SaveBinaryOutput error: %v", err)
	}
	if urlPath != "/outputs/other/file.bin" {
		t.Errorf("urlPath = %q", urlPath)
	}
}
