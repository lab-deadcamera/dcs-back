package file

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// TestDecodeImageByContent verifies that a JPEG stored with a ".png" extension
// still decodes — thumbnails must not depend on the file extension.
func TestDecodeImageByContent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "photo.png") // JPEG bytes under a .png name
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := decodeImage(path)
	if err != nil {
		t.Fatalf("decodeImage: %v", err)
	}
	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 4 {
		t.Fatalf("unexpected bounds: %v", got.Bounds())
	}
}

// TestGenerateThumbnailMislabeled verifies the full thumbnail pipeline works
// for a file whose extension doesn't match its content.
func TestGenerateThumbnailMislabeled(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}

	dir := t.TempDir()
	sub := "images/photo.png" // JPEG bytes under a .png name
	full := filepath.Join(dir, sub)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	s := &Store{uploadDir: dir}
	thumb, err := s.GenerateThumbnail(sub, 300, 300)
	if err != nil {
		t.Fatalf("GenerateThumbnail: %v", err)
	}
	if _, err := os.Stat(thumb); err != nil {
		t.Fatalf("thumbnail missing: %v", err)
	}
}
