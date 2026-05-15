package generators

import (
	"fmt"
	"strings"
	"time"
)

var nameModelSeedream = "dreamina-seedream-4-pro-251224"

type SeedreamGenerator struct {
	client seedanceAPI
}

type seedanceAPI interface {
	arkRequest(url, method string, body interface{}, apiKey string) (map[string]interface{}, error)
}

func NewSeedreamGenerator(outputsDir string) *SeedreamGenerator {
	return &SeedreamGenerator{
		client: NewSeedanceGenerator(outputsDir),
	}
}

func (g *SeedreamGenerator) Name() string { return nameModelSeedream }

func (g *SeedreamGenerator) Match(modelName string) bool {
	return strings.Contains(strings.ToLower(modelName), nameModelSeedream)
}

func (g *SeedreamGenerator) Generate(req *GeneratorRequest) (*GeneratorResult, error) {
	payload := map[string]interface{}{
		"model":           req.Model,
		"prompt":          compileContentText(req.Content),
		"size":            "2K",
		"response_format": "url",
		"watermark":       req.Watermark,
	}

	if req.Resolution != "" {
		payload["size"] = req.Resolution
	}

	var imageURLs []string
	for _, item := range req.Content {
		if item.Type == "image" && item.DataURL != "" {
			imageURLs = append(imageURLs, item.DataURL)
		}
	}
	if len(imageURLs) == 1 {
		payload["image"] = imageURLs[0]
	} else if len(imageURLs) > 1 {
		payload["image"] = imageURLs
	}

	result, err := g.client.arkRequest(req.BaseURL+req.Endpoint+"/images/generations", "POST", payload, req.APIKey)
	if err != nil {
		return nil, err
	}

	// Seedream is synchronous — extract image URLs from response
	var outputs []OutputResource
	if data, ok := result["data"].([]interface{}); ok {
		for _, d := range data {
			if entry, ok := d.(map[string]interface{}); ok {
				if url, ok := entry["url"].(string); ok && url != "" {
					outputs = append(outputs, OutputResource{
						URL:  url,
						Type: "image",
					})
				}
			}
		}
	}

	if len(outputs) == 0 {
		// fallback: check for single URL at top level
		for _, key := range []string{"url", "image_url", "data"} {
			if s, ok := result[key].(string); ok && s != "" {
				outputs = append(outputs, OutputResource{URL: s, Type: "image"})
				break
			}
		}
	}

	taskID, _ := result["id"].(string)
	if taskID == "" {
		taskID = fmt.Sprintf("seedream_%d", time.Now().UnixMilli())
	}

	return &GeneratorResult{
		TaskID:  taskID,
		Model:   req.Model,
		Status:  "succeeded",
		Outputs: outputs,
		Raw:     result,
	}, nil
}

func (g *SeedreamGenerator) GetStatus(taskID, apiKey, baseURL, endpoint string) (*GeneratorResult, error) {
	return nil, fmt.Errorf("seedream tasks complete synchronously, use the generate response directly")
}

func (g *SeedreamGenerator) CancelTask(taskID, apiKey, baseURL, endpoint string) error {
	return fmt.Errorf("seedream tasks cannot be cancelled")
}

// Ensure SeedanceGenerator satisfies the interface
var _ seedanceAPI = (*SeedanceGenerator)(nil)

/*
Seedream Generator — configuración del modelo (datos públicos de referencia)

Modelos que matchea:
  - dreamina-seedream-4-pro-251224   → texto a imagen (pro)
  - dreamina-seedream-4-0-250828    → texto a imagen

Base URLs (BytePlus ModelArk):
  BytePlus AP (default): https://ark.ap-southeast.bytepluses.com/api/v3
  Volcengine CN:         https://ark.cn-beijing.volces.com/api/v3

Endpoint:
  POST /images/generations

Auth:
  Header: Authorization: Bearer <ARK_API_KEY>
  Prefijo de key: "ark-"

Parámetros del payload:
  model         string   — nombre del modelo
  prompt        string   — descripción textual de la imagen
  size          string   — 2K (default), 1080p, 720p
  response_format string — "url" (default)
  watermark     bool     — incluir marca de agua
  seed          int      — semilla para reproducibilidad (opcional)
  image[]       array    — imágenes de referencia (base64 opcional)

Output:
  type: image (png/jpeg)
  URL de descarga: https://ark-*.bytepluses.com/... (expira en 24h)
  Las imágenes generadas se guardan en trustedAssets (memoria, expira 30 días)

Rate limits:
  Varían por tier de suscripción. Consultar console.byteplus.com
*/

