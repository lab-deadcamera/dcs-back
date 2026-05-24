package calculators

import (
	"math"
	"strings"

	"dcs-back-v0/internal/modules/studio"
)

// SeedanceCalculator estimates cost for Dreamina Seedance using the official
// BytePlus token-based formula extracted from the Price Calculator form:
//
//	TokenUsage = (inputDuration + outputDuration) x width x height x FPS / 1024
//	MinTokens  = if input exists: (MAX(ceil(output*2/3), input) + output) x W x H x FPS / 1024
//	             else: same as TokenUsage
//	Price      = UnitPrice / 1,000,000 x MAX(TokenUsage, MinTokens)
//
// Unit prices (USD per million tokens):
//
//	dreamina-seedance-2-0-260128:
//	  With Video: 480p/720p = 4.30 | 1080p = 4.70
//	  Without:    480p/720p = 7.00 | 1080p = 7.70
//
//	dreamina-seedance-2-0-fast-260128:
//	  With Video: 480p/720p = 3.30
//	  Without:    480p = 5.80 | 720p = 6.60
type SeedanceCalculator struct{}

func NewSeedanceCalculator() *SeedanceCalculator {
	return &SeedanceCalculator{}
}

func (c *SeedanceCalculator) Name() string { return "seedance" }

func (c *SeedanceCalculator) Match(modelName string) bool {
	lower := strings.ToLower(modelName)
	return strings.Contains(lower, "dreamina-seedance")
}

func (c *SeedanceCalculator) CalculateFromResponse(raw interface{}, req *studio.GeneratorRequest) (float64, bool) {
	return 0, false
}

func (c *SeedanceCalculator) CalculateEstimated(req *studio.GeneratorRequest) float64 {
	unitPrice := seedanceUnitPrice(req)
	if unitPrice == 0 {
		return 0
	}

	fps := 24.0
	width, height := seedanceDimensions(req.Resolution, req.Ratio)

	// Detect if input video/reference media is present
	hasInputVideo := false
	inputDuration := req.InputDuration
if inputDuration < 0 { inputDuration = 0 }
	for _, item := range req.Content {
		if item.Type == "video" || item.Type == "image" {
			hasInputVideo = true
			break
		}
	}

	outputDuration := float64(req.Duration)
	if outputDuration <= 0 {
		outputDuration = 5
	}

	// Token usage formula
	tokenUsage := (inputDuration + outputDuration) * width * height * fps / 1024

	// Minimum token usage
	var minTokens float64
	if hasInputVideo {
		// If input duration < output * 2/3, use ceil(output * 2/3) instead
		effectiveInput := math.Ceil(outputDuration * 2.0 / 3.0)
		if inputDuration > effectiveInput {
			effectiveInput = inputDuration
		}
		minTokens = (effectiveInput + outputDuration) * width * height * fps / 1024
	} else {
		minTokens = tokenUsage
	}

	usedTokens := tokenUsage
	if minTokens > usedTokens {
		usedTokens = minTokens
	}

	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	return unitPrice / 1_000_000 * usedTokens * float64(quantity)
}

func (c *SeedanceCalculator) NeedsBackgroundCalc() bool { return false }

// seedanceUnitPrice returns the price per million tokens for the given model config.
func seedanceUnitPrice(req *studio.GeneratorRequest) float64 {
	isFast := strings.Contains(strings.ToLower(req.Model), "fast")

	// Detect if input video/reference media is present
	hasInputVideo := false
	for _, item := range req.Content {
		if item.Type == "video" || item.Type == "image" {
			hasInputVideo = true
			break
		}
	}

	switch {
	case !isFast && hasInputVideo:
		switch req.Resolution {
		case "1080p":
			return 4.70
		default: // 480p, 720p
			return 4.30
		}
	case !isFast && !hasInputVideo:
		switch req.Resolution {
		case "1080p":
			return 7.70
		default: // 480p, 720p
			return 7.00
		}
	case isFast && hasInputVideo:
		return 3.30
	case isFast && !hasInputVideo:
		switch req.Resolution {
		case "720p":
			return 6.60
		default: // 480p
			return 5.80
		}
	default:
		return 0
	}
}

// seedanceDimensions returns (width, height) for a given resolution and ratio.
func seedanceDimensions(resolution, ratio string) (float64, float64) {
	// Default to 16:9 if ratio is empty or unrecognized
	isVertical := strings.Contains(ratio, "9:16")
	isSquare := ratio == "1:1"
	is43 := ratio == "4:3"
	is34 := ratio == "3:4"
	is219 := ratio == "21:9"

	switch resolution {
	case "480p":
		switch {
		case isVertical:
			return 480, 854
		case isSquare:
			return 640, 640
		case is43:
			return 640, 480
		case is34:
			return 480, 640
		case is219:
			return 1120, 480
		default: // 16:9
			return 854, 480
		}
	case "1080p":
		switch {
		case isVertical:
			return 1080, 1920
		case isSquare:
			return 1080, 1080
		case is43:
			return 1440, 1080
		case is34:
			return 1080, 1440
		case is219:
			return 2520, 1080
		default: // 16:9
			return 1920, 1080
		}
	default: // 720p
		switch {
		case isVertical:
			return 720, 1280
		case isSquare:
			return 720, 720
		case is43:
			return 960, 720
		case is34:
			return 720, 960
		case is219:
			return 1680, 720
		default: // 16:9
			return 1280, 720
		}
	}
}
