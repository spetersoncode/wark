package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTicketKey(t *testing.T) {
	tests := []struct {
		key         string
		wantProject string
		wantNumber  int
		wantErr     bool
	}{
		// Valid full keys
		{"WEBAPP-42", "WEBAPP", 42, false},
		{"TEST-1", "TEST", 1, false},
		{"ABC123-999", "ABC123", 999, false},
		{"A-1", "A", 1, false},
		{"X1-100", "X1", 100, false},

		// Valid with lowercase (should be uppercased)
		{"webapp-42", "WEBAPP", 42, false},
		{"Test-1", "TEST", 1, false},

		// Valid with whitespace (should be trimmed)
		{"  WEBAPP-42  ", "WEBAPP", 42, false},

		// Just a number (for use with default project)
		{"42", "", 42, false},
		{"1", "", 1, false},
		{"999", "", 999, false},

		// Invalid keys
		{"invalid", "", 0, true},
		{"WEBAPP-", "", 0, true},
		{"-42", "", 0, true},
		{"WEBAPP-abc", "", 0, true},
		{"123-456", "", 0, true},         // project must start with letter
		{"WEBAPP--42", "", 0, true},      // double dash
		{"WEBAPP-42-extra", "", 0, true}, // too many parts
		{"", "", 0, true},
		{" ", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			project, number, err := ParseTicketKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidTicketKey)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantProject, project)
				assert.Equal(t, tt.wantNumber, number)
			}
		})
	}
}
