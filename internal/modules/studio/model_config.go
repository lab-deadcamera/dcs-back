package studio

import (
	"fmt"

	"dcs-back-v0/internal/modules/provider"
)

// enforceVideoLimits validates genReq.Quantity (number of videos requested
// per generation) and genReq.Duration (seconds) against the limits declared
// in the model's config JSONB column ({"min_videos": N, "max_videos": M,
// "min_duration": A, "max_duration": B}). Zero values mean "no limit
// configured" and are ignored. An unset quantity (<= 0) defaults to 1
// before checking the minimum.
func enforceVideoLimits(genReq *GeneratorRequest, cfg provider.ModelConfig) error {
	if genReq.Quantity <= 0 {
		genReq.Quantity = 1
	}
	if cfg.MinVideos > 0 && genReq.Quantity < cfg.MinVideos {
		return fmt.Errorf("quantity must be at least %d for model %s", cfg.MinVideos, genReq.Model)
	}
	if cfg.MaxVideos > 0 && genReq.Quantity > cfg.MaxVideos {
		return fmt.Errorf("quantity must be at most %d for model %s", cfg.MaxVideos, genReq.Model)
	}
	if cfg.MinDuration > 0 && genReq.Duration < cfg.MinDuration {
		return fmt.Errorf("duration must be at least %d seconds for model %s", cfg.MinDuration, genReq.Model)
	}
	if cfg.MaxDuration > 0 && genReq.Duration > cfg.MaxDuration {
		return fmt.Errorf("duration must be at most %d seconds for model %s", cfg.MaxDuration, genReq.Model)
	}
	return nil
}
