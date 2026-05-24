package calculators

import (
	"strings"

	"dcs-back-v0/internal/modules/studio"
)

// SeedreamCalculator estimates cost for BytePlus image generation models.
// Pricing is per image (not per token), varies by model.
// Official prices (USD/image):
//
//	seedream-5-0-lite-260128  → 0.035
//	seedream-4-5-251128       → 0.04
//	seedream-4-0-250828       → 0.03
//	seededit-3-0-i2i-250628   → 0.03
type SeedreamCalculator struct {
	models map[string]float64 // model name prefix → price per image
}

func NewSeedreamCalculator() *SeedreamCalculator {
	return &SeedreamCalculator{
		models: map[string]float64{
			"seedream-5-0-lite": 0.035,
			"seedream-4-5":      0.04,
			"dreamina-seedream": 0.04, // fallback for older naming
			"seedream-4-0":      0.03,
			"seededit-3-0":      0.03,
		},
	}
}

func (c *SeedreamCalculator) Name() string { return "seedream" }

func (c *SeedreamCalculator) Match(modelName string) bool {
	lower := strings.ToLower(modelName)
	for prefix := range c.models {
		if strings.Contains(lower, prefix) {
			return true
		}
	}
	return false
}

func (c *SeedreamCalculator) CalculateFromResponse(raw interface{}, req *studio.GeneratorRequest) (float64, bool) {
	return 0, false
}

func (c *SeedreamCalculator) CalculateEstimated(req *studio.GeneratorRequest) float64 {
	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	price := c.priceForModel(req.Model)
	return price * float64(quantity)
}

func (c *SeedreamCalculator) NeedsBackgroundCalc() bool { return false }

func (c *SeedreamCalculator) priceForModel(modelName string) float64 {
	lower := strings.ToLower(modelName)
	for prefix, price := range c.models {
		if strings.Contains(lower, prefix) {
			return price
		}
	}
	return 0.04 // default fallback
}
