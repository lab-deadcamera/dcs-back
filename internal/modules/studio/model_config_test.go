package studio

import (
	"strings"
	"testing"

	"dcs-back-v0/internal/modules/provider"
)

func TestEnforceVideoLimits(t *testing.T) {
	tests := []struct {
		name      string
		config    provider.ModelConfig
		quantity  int
		wantErr   bool
		wantQty   int
		errSubstr string
	}{
		{
			name:     "no config, unset quantity defaults to 1",
			config:   provider.ModelConfig{},
			quantity: 0,
			wantQty:  1,
		},
		{
			name:     "no config, quantity passes through",
			config:   provider.ModelConfig{},
			quantity: 5,
			wantQty:  5,
		},
		{
			name:     "within configured range",
			config:   provider.ModelConfig{MinVideos: 1, MaxVideos: 4},
			quantity: 3,
			wantQty:  3,
		},
		{
			name:      "below minimum",
			config:    provider.ModelConfig{MinVideos: 2, MaxVideos: 4},
			quantity:  1,
			wantErr:   true,
			errSubstr: "at least 2",
		},
		{
			name:      "above maximum",
			config:    provider.ModelConfig{MinVideos: 1, MaxVideos: 4},
			quantity:  5,
			wantErr:   true,
			errSubstr: "at most 4",
		},
		{
			name:     "only max configured, ok",
			config:   provider.ModelConfig{MaxVideos: 4},
			quantity: 4,
			wantQty:  4,
		},
		{
			name:      "only max configured, exceeded",
			config:    provider.ModelConfig{MaxVideos: 2},
			quantity:  3,
			wantErr:   true,
			errSubstr: "at most 2",
		},
		{
			name:     "only min configured, ok",
			config:   provider.ModelConfig{MinVideos: 2},
			quantity: 2,
			wantQty:  2,
		},
		{
			name:      "unset quantity below configured minimum",
			config:    provider.ModelConfig{MinVideos: 2},
			quantity:  0,
			wantErr:   true,
			errSubstr: "at least 2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &GeneratorRequest{
				Model:    "dreamina-seedance-2-5-260628",
				Quantity: tc.quantity,
				Content:  []ContentItem{{Type: "text", Text: "a prompt"}},
			}
			err := enforceVideoLimits(req, tc.config)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if req.Quantity != tc.wantQty {
				t.Errorf("quantity = %d, want %d", req.Quantity, tc.wantQty)
			}
		})
	}
}
