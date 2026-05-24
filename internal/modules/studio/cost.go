package studio

// CostCalculator defines how to compute the estimated cost of a generation.
// Different models may use different strategies:
//   - Extract cost directly from the API response
//   - Call an external pricing API
//   - Compute locally based on request parameters
type CostCalculator interface {
	// Match returns true if this calculator handles the given model name.
	Match(modelName string) bool
	// CalculateFromResponse extracts cost directly from an API response.
	// Returns (cost, true) if the API included cost info, (0, false) otherwise.
	CalculateFromResponse(raw interface{}, req *GeneratorRequest) (float64, bool)
	// CalculateEstimated computes cost from request parameters (fallback).
	// Used when the API does not return cost info.
	CalculateEstimated(req *GeneratorRequest) float64
	// NeedsBackgroundCalc returns true if CalculateEstimated should run
	// in background (e.g., calls an external pricing API).
	NeedsBackgroundCalc() bool
	// Name returns a human-readable name for this calculator.
	Name() string
}

// CostResult wraps the calculated cost info for logging.
type CostResult struct {
	EstimatedCost float64
	Source        string // "api_response", "calculator", "pending"
}
