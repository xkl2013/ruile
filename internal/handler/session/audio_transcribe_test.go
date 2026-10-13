package session

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestConfiguredAgentASRModelIDRequiresExplicitEnabledConfig(t *testing.T) {
	tests := []struct {
		name  string
		agent *types.CustomAgent
		want  string
	}{
		{
			name: "enabled and configured",
			agent: &types.CustomAgent{
				Config: types.CustomAgentConfig{
					AudioUploadEnabled: true,
					ASRModelID:         "asr-model",
				},
			},
			want: "asr-model",
		},
		{
			name: "disabled even with model",
			agent: &types.CustomAgent{
				Config: types.CustomAgentConfig{ASRModelID: "asr-model"},
			},
		},
		{
			name: "enabled without model",
			agent: &types.CustomAgent{
				Config: types.CustomAgentConfig{AudioUploadEnabled: true},
			},
		},
		{name: "missing agent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := configuredAgentASRModelID(test.agent); got != test.want {
				t.Fatalf("configuredAgentASRModelID() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDefaultASRModelIDPrefersActiveDefault(t *testing.T) {
	models := []*types.Model{
		{ID: "inactive", Type: types.ModelTypeASR, Status: "downloading"},
		{ID: "fallback", Type: types.ModelTypeASR, Status: types.ModelStatusActive},
		{ID: "default", Type: types.ModelTypeASR, Status: types.ModelStatusActive, IsDefault: true},
	}

	if got := defaultASRModelID(models); got != "default" {
		t.Fatalf("defaultASRModelID() = %q, want %q", got, "default")
	}
}
