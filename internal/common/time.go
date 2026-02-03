package common

import (
	"fmt"
	"time"
)

// FormatAge returns a human-readable age string for a timestamp.
// Examples: "just now", "5m ago", "3h ago", "2d ago"
func FormatAge(t time.Time) string {
	return FormatDuration(time.Since(t))
}

// FormatDuration returns a human-readable string for a duration.
// Examples: "just now", "5m ago", "3h ago", "2d ago"
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}
