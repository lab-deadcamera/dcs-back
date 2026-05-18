package video

import (
	"dcs-back-v0/internal/studio"
)

type videoService struct {
	core *studio.Service
}

// NewService creates a Service adapter that satisfies video.Service.
func NewService(core *studio.Service) Service {
	return &videoService{core: core}
}

func (s *videoService) GenerateVideo(req *GenerateRequest) (*GenerateResponse, error) {
	unified := toStudioRequest(req)
	result, err := s.core.GenerateUnified(unified)
	if err != nil {
		return nil, err
	}
	return &GenerateResponse{
		TaskID:  result.TaskID,
		Model:   result.Model,
		Status:  result.Status,
		Outputs: toVideoResources(result.Outputs),
	}, nil
}

func (s *videoService) GetVideoStatus(taskID string) (*StatusResponse, error) {
	sr, err := s.core.GetStatusUnified(taskID)
	if err != nil {
		return nil, err
	}

	resp := &StatusResponse{
		Status:  sr.Status,
		Error:   sr.Error,
		Raw:     sr.Raw,
		Outputs: []OutputResource{},
	}

	if rawMap, ok := sr.Raw.(map[string]interface{}); ok {
		for _, key := range []string{"progress", "percentage", "task_progress"} {
			if v, exists := rawMap[key]; exists {
				resp.Progress = v
				break
			}
		}
	}
	for _, o := range sr.Outputs {
		resp.Outputs = append(resp.Outputs, OutputResource{
			URL:      o.URL,
			LocalURL: o.LocalURL,
		})
	}
	return resp, nil
}

func (s *videoService) CancelVideoTask(taskID string) error {
	return s.core.CancelTask(taskID)
}

func (s *videoService) PreviewVideoPayload(req *GenerateRequest) (*PreviewPayloadResponse, error) {
	unified := toStudioRequest(req)
	result, err := s.core.PreviewPayload(unified)
	if err != nil {
		return nil, err
	}
	return &PreviewPayloadResponse{
		Model:       result.Model,
		Endpoint:    result.Endpoint,
		Payload:     result.Payload,
		ContentType: result.ContentType,
	}, nil
}

// ─── Type converters ──────────────────────────────────────────

func toStudioRequest(req *GenerateRequest) *studio.StudioGenerateRequest {
	content := make([]studio.ContentItem, len(req.Content))
	for i, item := range req.Content {
		content[i] = studio.ContentItem{
			Type: item.Type,
			Text: item.Text,
			Name: item.Name,
			ID:   item.ID,
		}
	}
	return &studio.StudioGenerateRequest{
		Model:         req.Model,
		Content:       content,
		Ratio:         req.Ratio,
		Duration:      req.Duration,
		CameraFixed:   req.CameraFixed,
		Seed:          req.Seed,
		Quality:       req.Quality,
		Quantity:      req.Quantity,
		Watermark:     req.Watermark,
		Resolution:    req.Resolution,
		GenerateAudio: req.GenerateAudio,
		ProjectID:     req.ProjectID,
		SceneID:       req.SceneID,
		SceneCode:     req.SceneCode,
		TakeNumber:    req.TakeNumber,
		UserID:        req.UserID,
	}
}

func toVideoResources(src []studio.OutputResource) []OutputResource {
	out := make([]OutputResource, len(src))
	for i, o := range src {
		out[i] = OutputResource{
			URL:      o.URL,
			LocalURL: o.LocalURL,
		}
	}
	return out
}

// Ensure videoService satisfies Service at compile time.
var _ Service = (*videoService)(nil)
