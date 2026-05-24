// Package calculators provides CostCalculator implementations for each model.
package calculators

import "dcs-back-v0/internal/modules/studio"

// DefaultCalculator returns zero cost for models without specific pricing.
type DefaultCalculator struct{}

func NewDefaultCalculator() *DefaultCalculator {
	return &DefaultCalculator{}
}

func (c *DefaultCalculator) Name() string { return "default" }
func (c *DefaultCalculator) Match(_ string) bool { return true }
func (c *DefaultCalculator) CalculateFromResponse(_ interface{}, _ *studio.GeneratorRequest) (float64, bool) {
	return 0, false
}
func (c *DefaultCalculator) CalculateEstimated(_ *studio.GeneratorRequest) float64 {
	return 0
}
func (c *DefaultCalculator) NeedsBackgroundCalc() bool { return false }
