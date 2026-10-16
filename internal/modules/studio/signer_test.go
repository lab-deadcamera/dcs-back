package studio

import (
	"strings"
	"testing"
)

func TestSanitizeAssetName(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"short name unchanged", "scene-01.mp4"},
		{"exactly 64 chars", strings.Repeat("a", 60) + ".mp4"},
		{
			"real long video filename",
			"SWIPE RIGHT_THE SERIES_SC188_a0c53c77_T1_Laura_cgt-20261015_1792072270379.mp4",
		},
		{"long name with dots", strings.Repeat("x", 80) + ".test.png"},
		{"no extension", strings.Repeat("y", 100)},
		{"extension beyond the limit", strings.Repeat("z", 64) + ".mp4"},
	}
	for _, tc := range tests {
		got := sanitizeAssetName(tc.in)
		if len(got) > assetNameMaxLen {
			t.Errorf("%s: sanitizeAssetName(%q) returned %d bytes (max %d)", tc.name, tc.in, len(got), assetNameMaxLen)
		}
		// Names within the limit must pass through untouched.
		if len(tc.in) <= assetNameMaxLen && got != tc.in {
			t.Errorf("%s: sanitizeAssetName(%q) = %q, want unchanged", tc.name, tc.in, got)
		}
		// Truncated names keep the file extension so the library entry stays identifiable.
		if len(tc.in) > assetNameMaxLen {
			ext := ""
			if i := strings.LastIndex(tc.in, "."); i > 0 && i < len(tc.in)-1 && i < assetNameMaxLen {
				ext = tc.in[i:]
			}
			if ext != "" && !strings.HasSuffix(got, ext) {
				t.Errorf("%s: sanitizeAssetName(%q) = %q, want suffix %q", tc.name, tc.in, got, ext)
			}
			if !strings.HasPrefix(got, tc.in[:1]) {
				t.Errorf("%s: sanitizeAssetName(%q) = %q, want prefix of the original", tc.name, tc.in, got)
			}
		}
	}
}
