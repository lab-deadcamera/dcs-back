package studio

import "strings"

// GalleryModels lists all model names that perform automatic gallery sync.
// Add a model name here (full or partial match) to enable gallery sync.
// The sync happens in GallerySyncContent before generation:
//   - non-text content items are uploaded to the model's BytePlus asset library
//   - character assets are grouped by character name
//   - DataURL is replaced with the model-specific reference URI
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

// BuildReferenceURI construye la URI de referencia según el tipo de modelo.
// Cada modelo puede requerir un formato distinto para referenciar assets:
//
//	Modelos "gallery" (BytePlus Seedance) → "asset://<AssetID>"
//	                                                         El API de BytePlus entiende este formato
//	                                                         y resuelve el asset internamente.
//
//	Otros modelos                          → assetURL (CDN directa)
//	                                                         Usan la URL pública del asset.
//
// Agregar nuevos formatos aquí cuando se integren modelos con requisitos diferentes.
func BuildReferenceURI(modelName string, assetID, assetURL string) string {
	if IsGalleryModel(modelName) {
		// Los modelos gallery (BytePlus) usan asset://<id> porque el API
		// resuelve internamente el asset desde la galería privada del modelo.
		return "asset://" + assetID
	}
	// Para el resto de modelos se usa la URL directa del CDN.
	return assetURL
}
