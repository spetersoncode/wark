package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"just now - 0 seconds", 0, "just now"},
		{"just now - 30 seconds", 30 * time.Second, "just now"},
		{"just now - 59 seconds", 59 * time.Second, "just now"},
		{"minutes - 1 minute", 1 * time.Minute, "1m ago"},
		{"minutes - 5 minutes", 5 * time.Minute, "5m ago"},
		{"minutes - 59 minutes", 59 * time.Minute, "59m ago"},
		{"hours - 1 hour", 1 * time.Hour, "1h ago"},
		{"hours - 3 hours", 3 * time.Hour, "3h ago"},
		{"hours - 23 hours", 23 * time.Hour, "23h ago"},
		{"days - 1 day", 24 * time.Hour, "1d ago"},
		{"days - 2 days", 48 * time.Hour, "2d ago"},
		{"days - 30 days", 30 * 24 * time.Hour, "30d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp := time.Now().Add(-tt.duration)
			got := FormatAge(timestamp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, "just now"},
		{"seconds", 45 * time.Second, "just now"},
		{"minute boundary", 60 * time.Second, "1m ago"},
		{"minutes", 15 * time.Minute, "15m ago"},
		{"hour boundary", 60 * time.Minute, "1h ago"},
		{"hours", 12 * time.Hour, "12h ago"},
		{"day boundary", 24 * time.Hour, "1d ago"},
		{"days", 7 * 24 * time.Hour, "7d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}
