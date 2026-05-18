package studio

import "strings"

// GalleryModels lists all model names that perform automatic gallery sync.
// Add a model name here (full or partial match) to enable gallery sync.
// The sync happens in GallerySyncContent before generation:
//   - non-text content items are uploaded to the model's BytePlus asset library
//   - character assets are grouped by character name
//   - DataURL is replaced with asset://<AssetId>
var GalleryModels = []string{
	"dreamina-seedance-2-0-gallery",
}

// IsGalleryModel returns true if the given model name matches any registered gallery model.
func IsGalleryModel(modelName string) bool {
	lower := strings.ToLower(modelName)
	for _, m := range GalleryModels {
		if strings.Contains(lower, strings.ToLower(m)) {
			return true
		}
	}
	return false
}
