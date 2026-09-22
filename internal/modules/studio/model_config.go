package studio

import (
	"fmt"
	"slices"

	"dcs-back-v0/internal/modules/provider"
)

// enforceVideoLimits validates genReq.Quantity (number of videos requested
// per generation), genReq.Duration (seconds), the aspect ratio and the
// resolution against the limits declared in the model's config JSONB column
// ({"min_videos": N, "max_videos": M, "min_duration": A, "max_duration": B,
// "aspect_ratios": [...], "resolutions": [...]}). Zero values / empty slices
// mean "no limit configured" and are ignored. An unset quantity (<= 0)
// defaults to 1 before checking the minimum.
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
	if len(cfg.AspectRatios) > 0 && !slices.Contains(cfg.AspectRatios, genReq.Ratio) {
		return fmt.Errorf("aspect ratio %q is not supported by model %s", genReq.Ratio, genReq.Model)
	}
	if len(cfg.Resolutions) > 0 && !slices.Contains(cfg.Resolutions, genReq.Resolution) {
		return fmt.Errorf("resolution %q is not supported by model %s", genReq.Resolution, genReq.Model)
	}
	return nil
}
