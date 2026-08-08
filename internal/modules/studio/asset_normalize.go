package studio

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

// BytePlus validation limits for uploaded assets. Images outside these
// bounds are rejected by CreateAsset with the errors seen in model_assets
// ("Height must be between 300px and 6000px", "Aspect ratio must be between
// 0.4 and 2.5").
const (
	BytePlusMinHeight = 300
	BytePlusMaxHeight = 6000
	BytePlusMinAspect = 0.4
	BytePlusMaxAspect = 2.5
)

// imageDims reads an image's pixel dimensions from a local file.
func imageDims(path string) (w, h int, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

// imageNeedsFix reports whether the dimensions violate BytePlus limits.
func imageNeedsFix(w, h int) bool {
	if w <= 0 || h <= 0 {
		return false
	}
	if h < BytePlusMinHeight || h > BytePlusMaxHeight {
		return true
	}
	aspect := float64(w) / float64(h)
	return aspect < BytePlusMinAspect || aspect > BytePlusMaxAspect
}

// normalizeImage reads the image at srcPath and, if it violates BytePlus
// limits, returns JPEG bytes of a normalized copy. Returns (nil, nil) when no
// fix is needed. `aspectFix` selects the strategy for out-of-range aspect
// ratios: "crop" trims the edges (loses content), anything else pads with
// bars (preserves all content).
func normalizeImage(srcPath, aspectFix string) ([]byte, error) {
	w, h, ok := imageDims(srcPath)
	if !ok || !imageNeedsFix(w, h) {
		return nil, nil
	}

	src, err := imaging.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, fmt.Errorf("image has invalid bounds")
	}

	// 1. Height out of range → scale to keep within [min, max].
	if srcH < BytePlusMinHeight {
		src = imaging.Resize(src, 0, BytePlusMinHeight, imaging.Lanczos)
	} else if srcH > BytePlusMaxHeight {
		src = imaging.Resize(src, 0, BytePlusMaxHeight, imaging.Lanczos)
	}

	// 2. Aspect ratio out of range → pad (bars) or crop.
	// Recompute from the actual (possibly resized) bounds.
	aspect := float64(src.Bounds().Dx()) / float64(src.Bounds().Dy())
	if aspect < BytePlusMinAspect {
		// Too tall: target width for the same height at min aspect.
		targetW := int(float64(src.Bounds().Dy()) * BytePlusMinAspect)
		if aspectFix == "crop" {
			src = imaging.CropCenter(src, targetW, src.Bounds().Dy())
		} else {
			// Pad with black bars: canvas at the valid ratio, image pasted centered.
			canvas := imaging.New(targetW, src.Bounds().Dy(), color.Black)
			src = imaging.PasteCenter(canvas, src)
		}
	} else if aspect > BytePlusMaxAspect {
		// Too wide: target height for the same width at max aspect.
		targetH := int(float64(src.Bounds().Dx()) / BytePlusMaxAspect)
		if aspectFix == "crop" {
			src = imaging.CropCenter(src, src.Bounds().Dx(), targetH)
		} else {
			canvas := imaging.New(src.Bounds().Dx(), targetH, color.Black)
			src = imaging.PasteCenter(canvas, src)
		}
	}

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, src, imaging.JPEG, imaging.JPEGQuality(90)); err != nil {
		return nil, fmt.Errorf("failed to encode normalized image: %w", err)
	}
	return buf.Bytes(), nil
}

// isImageMime reports whether a MIME type is an image (jpg/png/webp/gif/…).
func isImageMime(mime string) bool {
	return strings.HasPrefix(mime, "image/")
}
