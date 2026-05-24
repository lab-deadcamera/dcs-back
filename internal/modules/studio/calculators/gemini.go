package calculators

import (
	"strings"

	"dcs-back-v0/internal/modules/studio"
)

// GeminiCalculator is a no-cost calculator for Gemini Nano models (free tier).
type GeminiCalculator struct{}

func NewGeminiCalculator() *GeminiCalculator {
	return &GeminiCalculator{}
}

func (c *GeminiCalculator) Name() string { return "gemini" }

func (c *GeminiCalculator) Match(modelName string) bool {
	return strings.Contains(strings.ToLower(modelName), "gemini")
}

func (c *GeminiCalculator) CalculateFromResponse(raw interface{}, req *studio.GeneratorRequest) (float64, bool) {
	return 0, false
}

func (c *GeminiCalculator) CalculateEstimated(req *studio.GeneratorRequest) float64 {
	return 0
}

func (c *GeminiCalculator) NeedsBackgroundCalc() bool { return false }
