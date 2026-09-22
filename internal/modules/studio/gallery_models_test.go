package studio

import "testing"

func TestGalleryOwnerModel(t *testing.T) {
	owner := "dreamina-seedance-2-0-gallery"
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"gallery 2.0", "dreamina-seedance-2-0-gallery", owner},
		{"gallery 2.5 shares owner", "dreamina-seedance-2-5-260628", owner},
		{"gallery 2.5 fast variant", "dreamina-seedance-2-5-260628-fast", owner},
		{"non-gallery model stays itself", "some-other-model", "some-other-model"},
	}
	for _, tc := range tests {
		if got := GalleryOwnerModel(tc.got); got != tc.want {
			t.Errorf("GalleryOwnerModel(%q) = %q, want %q", tc.got, got, tc.want)
		}
	}
}

func TestGalleryModels(t *testing.T) {
	for _, m := range GalleryModels {
		if !IsGalleryModel(m) {
			t.Errorf("IsGalleryModel(%q) = false, want true", m)
		}
	}
	if IsGalleryModel("some-other-model") {
		t.Error("IsGalleryModel(some-other-model) = true, want false")
	}
}