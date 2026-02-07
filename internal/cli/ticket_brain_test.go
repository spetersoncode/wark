package cli

import (
	"testing"

	"github.com/spetersoncode/wark/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBrainSpec(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		wantBrain *models.Brain
		wantErr   bool
	}{
		{
			name: "model brain",
			spec: "model:sonnet",
			wantBrain: &models.Brain{
				Type:  "model",
				Value: "sonnet",
			},
			wantErr: false,
		},
		{
			name: "tool brain",
			spec: "tool:claude-code",
			wantBrain: &models.Brain{
				Type:  "tool",
				Value: "claude-code",
			},
			wantErr: false,
		},
		{
			name: "model brain with opus",
			spec: "model:opus",
			wantBrain: &models.Brain{
				Type:  "model",
				Value: "opus",
			},
			wantErr: false,
		},
		{
			name: "model brain with qwen",
			spec: "model:qwen",
			wantBrain: &models.Brain{
				Type:  "model",
				Value: "qwen",
			},
			wantErr: false,
		},
		{
			name:      "missing colon",
			spec:      "modelsonnet",
			wantBrain: nil,
			wantErr:   true,
		},
		{
			name:      "missing value",
			spec:      "model:",
			wantBrain: nil,
			wantErr:   true,
		},
		{
			name:      "invalid type",
			spec:      "invalid:sonnet",
			wantBrain: nil,
			wantErr:   true,
		},
		{
			name:      "empty spec",
			spec:      "",
			wantBrain: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			brain, err := parseBrainSpec(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, brain)
			} else {
				require.NoError(t, err)
				require.NotNil(t, brain)
				assert.Equal(t, tt.wantBrain.Type, brain.Type)
				assert.Equal(t, tt.wantBrain.Value, brain.Value)
			}
		})
	}
}

// Integration tests are covered by the database tests in ticket_brain_test.go
// and can be manually tested using the CLI commands.
