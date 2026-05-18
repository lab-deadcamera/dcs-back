package video

import (
	"regexp"
	"strings"
)

// ValidRatios lists supported aspect ratios for video generation.
var ValidRatios = map[string]bool{
	"16:9": true, "9:16": true, "1:1": true, "4:3": true, "3:4": true,
	"21:9": true, "9:21": true, "2.35:1": true,
}

// ValidResolutionsVideo lists supported video resolutions.
var ValidResolutionsVideo = map[string]bool{
	"480p": true, "720p": true, "1080p": true,
}

// IsFastModel returns true if the model name contains "fast".
func IsFastModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "fast")
}

// VideoURLPattern matches HTTP(S) URLs for video file discovery.
var VideoURLPattern = regexp.MustCompile(`^https?://`)

// SafeSuffix returns the last 8 characters of a task ID for use in filenames.
func SafeSuffix(taskID string) string {
	if len(taskID) >= 8 {
		return taskID[len(taskID)-8:]
	}
	return taskID
}
